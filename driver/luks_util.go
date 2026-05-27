/*
Copyright 2019 linkyard ag
Copyright cloudscale.ch

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package driver

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"
)

const (
	// LuksEncryptedAttribute is used to pass the information if the volume should be
	// encrypted with luks to `NodeStageVolume`
	LuksEncryptedAttribute = DriverName + "/luks-encrypted"

	// LuksCipherAttribute is used to pass the information about the luks encryption
	// cypher to `NodeStageVolume`
	LuksCipherAttribute = DriverName + "/luks-cipher"

	// LuksKeySizeAttribute is used to pass the information about the luks key size
	// to `NodeStageVolume`
	LuksKeySizeAttribute = DriverName + "/luks-key-size"

	// LuksKeyAttribute is the key of the luks key used in the map of secrets passed from the CO
	LuksKeyAttribute = "luksKey"
)

type VolumeLifecycle string

const (
	VolumeLifecycleNodeStageVolume     VolumeLifecycle = "NodeStageVolume"
	VolumeLifecycleNodePublishVolume   VolumeLifecycle = "NodePublishVolume"
	VolumeLifecycleNodeUnstageVolume   VolumeLifecycle = "NodeUnstageVolume"
	VolumeLifecycleNodeUnpublishVolume VolumeLifecycle = "NodeUnpublishVolume"
)

type LuksContext struct {
	EncryptionEnabled bool
	EncryptionKey     string
	EncryptionCipher  string
	EncryptionKeySize string
	VolumeName        string
	VolumeLifecycle   VolumeLifecycle
}

func (ctx *LuksContext) validate() error {
	if !ctx.EncryptionEnabled {
		return nil
	}

	var appendFn = func(x string, xs string) string {
		if xs != "" {
			xs += "; "
		}
		xs += x
		return xs
	}

	errorMsg := ""
	if ctx.VolumeName == "" {
		errorMsg = appendFn("no volume name provided", errorMsg)
	}
	if ctx.EncryptionKey == "" {
		errorMsg = appendFn("no encryption key provided", errorMsg)
	}
	if ctx.EncryptionCipher == "" {
		errorMsg = appendFn("no encryption cipher provided", errorMsg)
	}
	if ctx.EncryptionKeySize == "" {
		errorMsg = appendFn("no encryption key size provided", errorMsg)
	}
	if errorMsg == "" {
		return nil
	}
	return errors.New(errorMsg)
}

func getLuksContext(secrets map[string]string, context map[string]string, lifecycle VolumeLifecycle) LuksContext {
	if context[LuksEncryptedAttribute] != "true" {
		return LuksContext{
			EncryptionEnabled: false,
			VolumeLifecycle:   lifecycle,
		}
	}

	luksKey := secrets[LuksKeyAttribute]
	luksCipher := context[LuksCipherAttribute]
	luksKeySize := context[LuksKeySizeAttribute]
	volumeName := context[PublishInfoVolumeName]

	return LuksContext{
		EncryptionEnabled: true,
		EncryptionKey:     luksKey,
		EncryptionCipher:  luksCipher,
		EncryptionKeySize: luksKeySize,
		VolumeName:        volumeName,
		VolumeLifecycle:   lifecycle,
	}
}

func luksFormat(source string, mkfsCmd string, mkfsArgs []string, ctx LuksContext, log *logrus.Entry) (err error) {
	cryptsetupCmd, err := getCryptsetupCmd()
	if err != nil {
		return err
	}
	filename, err := writeLuksKey(ctx.EncryptionKey, log)
	if err != nil {
		return err
	}

	defer func() {
		if e := os.Remove(filename); e != nil {
			log.Errorf("cannot delete temporary file %s: %s", filename, e.Error())
		}
	}()

	// initialize the luks partition
	cryptsetupArgs := []string{
		"-v",
		"--type=luks1",
		"--batch-mode",
		"--cipher", ctx.EncryptionCipher,
		"--key-size", ctx.EncryptionKeySize,
		"--key-file", filename,
		"luksFormat", source,
	}

	log.WithFields(logrus.Fields{
		"cmd":  cryptsetupCmd,
		"args": cryptsetupArgs,
	}).Info("executing cryptsetup luksFormat command")

	out, err := exec.Command(cryptsetupCmd, cryptsetupArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cryptsetup luksFormat failed: %v cmd: '%s %s' output: %q",
			err, cryptsetupCmd, strings.Join(cryptsetupArgs, " "), string(out))
	}

	// open the luks partition and set up a mapping
	opened, err := luksOpen(source, filename, ctx, log)
	if err != nil {
		return fmt.Errorf("luksOpen during format failed: %w", err)
	}

	if opened {
		defer func() {
			if e := luksClose(ctx.VolumeName, log); e != nil {
				log.Errorf("cannot close luks device: %s", e.Error())
				if err == nil {
					err = fmt.Errorf("luksClose after format failed: %w", e)
				}
			}
		}()
	}

	// replace the source volume with the mapped one in the arguments to mkfs
	mkfsNewArgs := make([]string, len(mkfsArgs))
	for i, elem := range mkfsArgs {
		if elem != source {
			mkfsNewArgs[i] = elem
		} else {
			mkfsArgs[i] = "/dev/mapper/" + ctx.VolumeName
		}
	}

	log.WithFields(logrus.Fields{
		"cmd":  mkfsCmd,
		"args": mkfsArgs,
	}).Info("executing format command")

	mkfsOut, err := exec.Command(mkfsCmd, mkfsArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("formatting disk failed: %v cmd: '%s %s' output: %q",
			err, mkfsCmd, strings.Join(mkfsArgs, " "), string(mkfsOut))
	}

	return nil
}

// prepares a luks-encrypted volume for mounting and returns the path of the mapped volume
func luksPrepareMount(source string, ctx LuksContext, log *logrus.Entry) (string, error) {
	filename, err := writeLuksKey(ctx.EncryptionKey, log)
	if err != nil {
		return "", err
	}
	defer func() {
		if err := os.Remove(filename); err != nil {
			log.Errorf("cannot delete temporary file %s: %s", filename, err.Error())
		}
	}()

	// The mapping is intentionally kept open until NodeUnstageVolume.
	if _, err := luksOpen(source, filename, ctx, log); err != nil {
		return "", err
	}
	return "/dev/mapper/" + ctx.VolumeName, nil
}

func luksClose(volume string, log *logrus.Entry) error {
	cryptsetupCmd, err := getCryptsetupCmd()
	if err != nil {
		return err
	}
	cryptsetupArgs := []string{"--batch-mode", "close", volume}

	log.WithFields(logrus.Fields{
		"cmd":  cryptsetupCmd,
		"args": cryptsetupArgs,
	}).Info("executing cryptsetup close command")

	out, err := exec.Command(cryptsetupCmd, cryptsetupArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("removing luks mapping failed: %v cmd: '%s %s' output: %q",
			err, cryptsetupCmd, strings.Join(cryptsetupArgs, " "), string(out))
	}
	return nil
}

// checks if the given volume is formatted by checking if it is a luks volume and
// if the luks volume, once opened, contains a filesystem
func isLuksVolumeFormatted(volume string, ctx LuksContext, log *logrus.Entry) (formatted bool, err error) {
	isLuks, err := isLuks(volume)
	if err != nil {
		return false, err
	}
	if !isLuks {
		return false, nil
	}

	filename, err := writeLuksKey(ctx.EncryptionKey, log)
	if err != nil {
		return false, err
	}
	defer func() {
		if e := os.Remove(filename); e != nil {
			log.Errorf("cannot delete temporary file %s: %s", filename, e.Error())
		}
	}()

	opened, err := luksOpen(volume, filename, ctx, log)
	if err != nil {
		return false, err
	}
	if opened {
		defer func() {
			if e := luksClose(ctx.VolumeName, log); e != nil {
				log.Errorf("cannot close luks device: %s", e.Error())
				if err == nil {
					err = fmt.Errorf("luksClose after format check failed: %w", e)
				}
			}
		}()
	}

	return isVolumeFormatted(volume, log)
}

// luksOpen ensures that /dev/mapper/<ctx.VolumeName> exists and is backed by
// the device referenced by `volume`. The boolean return value reports whether
// this call actually opened the mapping (true) or found an existing mapping
// it validated and reused (false). Callers that registered a deferred
// luksClose should gate it on this flag so they do not close a mapping they
// did not open.
func luksOpen(volume string, keyFile string, ctx LuksContext, log *logrus.Entry) (bool, error) {
	mapperPath := "/dev/mapper/" + ctx.VolumeName
	if _, statErr := os.Stat(mapperPath); statErr == nil {
		// A mapping with this name already exists. Confirm that it is
		// backed by the device we just resolved before reusing it.
		// Without this check, a stale mapping left over from a
		// previously-attached cloudscale volume can end up mounted over a
		// freshly attached volume. Once two staging paths share one
		// device-mapper minor, GetDeviceMountRefs refuses every subsequent
		// unstage and the node accumulates unrecoverable state.
		inactive, backing, err := validateExistingLuksMapping(ctx.VolumeName, volume, cryptsetupStatus)
		if err != nil {
			return false, err
		}
		if inactive {
			// The mapping was closed between Stat and status (another
			// goroutine raced us). Fall through to open it fresh.
			log.WithField("volume", volume).Info("luks mapping vanished between stat and status, opening fresh")
		} else {
			log.WithFields(logrus.Fields{
				"volume":  volume,
				"backing": backing,
			}).Info("luks volume is already open and backing device matches")
			return false, nil
		}
	} else if !os.IsNotExist(statErr) {
		return false, fmt.Errorf("stat %s: %w", mapperPath, statErr)
	}

	cryptsetupCmd, err := getCryptsetupCmd()
	if err != nil {
		return false, err
	}
	cryptsetupArgs := []string{
		"--batch-mode",
		"luksOpen",
		"--key-file", keyFile,
		volume, ctx.VolumeName,
	}
	log.WithFields(logrus.Fields{
		"cmd":  cryptsetupCmd,
		"args": cryptsetupArgs,
	}).Info("executing cryptsetup luksOpen command")
	out, err := exec.Command(cryptsetupCmd, cryptsetupArgs...).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("cryptsetup luksOpen failed: %v cmd: '%s %s' output: %q",
			err, cryptsetupCmd, strings.Join(cryptsetupArgs, " "), string(out))
	}
	return true, nil
}

// runs cryptsetup resize for a given volume (/dev/mapper/pvc-xyz)
func luksResize(volume string, log *logrus.Entry) error {
	cryptsetupCmd, err := getCryptsetupCmd()
	if err != nil {
		return err
	}
	cryptsetupArgs := []string{"--batch-mode", "resize", volume}

	log.WithFields(logrus.Fields{
		"cmd":  cryptsetupCmd,
		"args": cryptsetupArgs,
	}).Info("executing cryptsetup resize command")

	out, err := exec.Command(cryptsetupCmd, cryptsetupArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("cryptsetup resize failed: %v cmd: '%s %s' output: %q",
			err, cryptsetupCmd, strings.Join(cryptsetupArgs, " "), string(out))
	}
	return nil
}

// runs cryptsetup isLuks for a given volume
func isLuks(volume string) (bool, error) {
	cryptsetupCmd, err := getCryptsetupCmd()
	if err != nil {
		return false, err
	}
	cryptsetupArgs := []string{"--batch-mode", "isLuks", volume}

	// cryptsetup isLuks exits with code 0 if the target is a luks volume; otherwise it returns
	// a non-zero exit code which exec.Command interprets as an error
	_, err = exec.Command(cryptsetupCmd, cryptsetupArgs...).CombinedOutput()
	if err != nil {
		return false, nil
	}
	return true, nil
}

// cryptsetupStatusInfo holds the fields we parse from `cryptsetup status <name>`.
type cryptsetupStatusInfo struct {
	backing    string
	isLuks     bool
	isInactive bool
}

// parseCryptsetupStatus extracts the LUKS-ness, backing device, and
// active/inactive state from the output of `cryptsetup status <name>`.
// The first line is the state header, always one of:
//
//	"/dev/mapper/<name> is inactive."
//	"/dev/mapper/<name> is active and is in use."
//	"/dev/mapper/<name> is active."
//
// Subsequent lines (active case only) are "  key:  value", parsed in the
// loop below.
func parseCryptsetupStatus(out []byte) cryptsetupStatusInfo {
	var info cryptsetupStatusInfo
	scanner := bufio.NewScanner(bytes.NewReader(out))
	if !scanner.Scan() {
		return info
	}
	if strings.Contains(scanner.Text(), " is inactive") {
		info.isInactive = true
		return info
	}
	for scanner.Scan() {
		key, value, ok := strings.Cut(scanner.Text(), ":")
		if !ok {
			continue
		}
		switch strings.TrimSpace(key) {
		case "type":
			if strings.Contains(strings.ToLower(value), "luks") {
				info.isLuks = true
			}
		case "device":
			info.backing = strings.TrimSpace(value)
		}
	}
	return info
}

// validateExistingLuksMapping confirms that the existing /dev/mapper/<mapperName>
// is backed by the same block device as `volume`. It returns:
//   - (true, "", nil)         when the mapping is no longer active (a concurrent
//     close raced the stat); the caller should re-open.
//   - (false, backing, nil)   when the mapping is safe to reuse; `backing` is
//     the device path as reported by cryptsetup (useful for logging).
//   - (false, "", err)        when cryptsetup status fails, the mapping is
//     active but reports no backing device, or the backing device does not
//     match the resolved `volume`.
//
// The statusFn parameter exists so tests can substitute a fake without
// actually shelling out to cryptsetup.
func validateExistingLuksMapping(
	mapperName, volume string,
	statusFn func(string) (cryptsetupStatusInfo, error),
) (isInactive bool, backing string, err error) {
	info, err := statusFn(mapperName)
	if err != nil {
		return false, "", fmt.Errorf("luks mapping %s exists but cryptsetup status failed: %w",
			mapperName, err)
	}
	if info.isInactive {
		return true, "", nil
	}
	if info.backing == "" {
		return false, "", fmt.Errorf("luks mapping %s is active but cryptsetup status reported no backing device",
			mapperName)
	}
	expected, err := filepath.EvalSymlinks(volume)
	if err != nil {
		return false, "", fmt.Errorf("cannot resolve volume %s for luks mapping validation: %w",
			volume, err)
	}
	// Resolve both sides through EvalSymlinks: the cryptsetup we ship today
	// canonicalises the device path via realpath() at open time, but a future
	// cryptsetup could preserve the caller-supplied symlink. Comparing
	// resolved-against-resolved keeps the check correct either way.
	backingResolved, evalErr := filepath.EvalSymlinks(info.backing)
	if evalErr != nil {
		backingResolved = info.backing
	}
	if backingResolved != expected {
		return false, "", fmt.Errorf("luks mapping %s is backed by %s (resolved: %s), expected %s, refusing to reuse stale mapping",
			mapperName, info.backing, backingResolved, expected)
	}
	return false, info.backing, nil
}

// cryptsetupStatus runs `cryptsetup status <name>` and returns the parsed
// info. A mapping that is reported as inactive (the normal "no such mapping"
// case) is returned with info.isInactive == true and a nil error so callers
// can distinguish it from real failures — mirroring the sentinel pattern in
// ceph-csi's DeviceEncryptionStatus.
func cryptsetupStatus(name string) (cryptsetupStatusInfo, error) {
	cryptsetupCmd, err := getCryptsetupCmd()
	if err != nil {
		return cryptsetupStatusInfo{}, err
	}
	out, err := exec.Command(cryptsetupCmd, "status", name).CombinedOutput()
	info := parseCryptsetupStatus(out)
	if err != nil {
		if info.isInactive {
			return info, nil
		}
		return info, fmt.Errorf("cryptsetup status %s failed: %v output: %q",
			name, err, string(out))
	}
	return info, nil
}

// check is a given mapping under /dev/mapper is a luks volume
func isLuksMapping(volume string) (bool, string, error) {
	if !strings.HasPrefix(volume, "/dev/mapper/") {
		return false, "", nil
	}
	mappingName := volume[len("/dev/mapper/"):]
	info, err := cryptsetupStatus(mappingName)
	if err != nil {
		return false, mappingName, err
	}
	if info.isInactive {
		return false, "", nil
	}
	return info.isLuks, mappingName, nil
}

func getCryptsetupCmd() (string, error) {
	cryptsetupCmd := "cryptsetup"
	_, err := exec.LookPath(cryptsetupCmd)
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", fmt.Errorf("%q executable not found in $PATH", cryptsetupCmd)
		}
		return "", err
	}
	return cryptsetupCmd, nil
}

// writes the given luks encryption key to a temporary file and returns the name of the temporary
// file
func writeLuksKey(key string, log *logrus.Entry) (string, error) {
	if !checkTmpFs("/tmp") {
		return "", errors.New("temporary directory /tmp is not a tmpfs volume; refusing to write luks key to a volume backed by a disk")
	}
	tmpFile, err := os.CreateTemp("/tmp", "luks-")
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tmpFile.Close()
	}()

	_, err = tmpFile.WriteString(key)
	if err != nil {
		log.WithField("tmp_file", tmpFile.Name()).Warnf("Unable to write luks key file: %s", err.Error())
		return "", err
	}
	return tmpFile.Name(), nil
}

// makes sure that the given directory is a tmpfs
func checkTmpFs(dir string) bool {
	out, err := exec.Command("sh", "-c", "df -T "+dir+" | tail -n1 | awk '{print $2}'").CombinedOutput()
	if err != nil {
		return false
	}
	if len(out) == 0 {
		return false
	}
	return strings.TrimSpace(string(out)) == "tmpfs"
}
