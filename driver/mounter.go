/*
Copyright cloudscale.ch
Copyright 2018 DigitalOcean

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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
)

type findmntResponse struct {
	FileSystems []fileSystem `json:"filesystems"`
}

type fileSystem struct {
	Source		string `json:"source"`
	Target      string `json:"target"`
	Propagation string `json:"propagation"`
	FsType      string `json:"fstype"`
	Options     string `json:"options"`
}

// Mounter is responsible for formatting and mounting volumes
type Mounter interface {
	// Format formats the source with the given filesystem type
	Format(source, fsType string, luksContext LuksContext) error

	// Mount mounts source to target with the given fstype and options.
	Mount(source, target, fsType string, isBind bool, luksContext LuksContext, options ...string) error

	// Unmount unmounts the given target
	Unmount(target string, isStaging bool) error

	// IsFormatted checks whether the source device is formatted or not. It
	// returns true if the source device is already formatted.
	IsFormatted(source string, luksContext LuksContext) (bool, error)

	// IsMounted checks whether the target path is a correct mount (i.e:
	// propagated). It returns true if it's mounted. An error is returned in
	// case of system errors or if it's mounted incorrectly.
	IsMounted(target string) (bool, error)
}

// TODO(arslan): this is Linux only for now. Refactor this into a package with
// architecture specific code in the future, such as mounter_darwin.go,
// mounter_linux.go, etc..
type mounter struct {
	log *logrus.Entry
}

// newMounter returns a new mounter instance
func newMounter(log *logrus.Entry) *mounter {
	return &mounter{
		log: log,
	}
}

func (m *mounter) Format(source, fsType string, luksContext LuksContext) error {
	mkfsCmd := fmt.Sprintf("mkfs.%s", fsType)

	_, err := exec.LookPath(mkfsCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return fmt.Errorf("%q executable not found in $PATH", mkfsCmd)
		}
		return err
	}

	mkfsArgs := []string{}

	if fsType == "" {
		return errors.New("fs type is not specified for formatting the volume")
	}

	if source == "" {
		return errors.New("source is not specified for formatting the volume")
	}

	mkfsArgs = append(mkfsArgs, source)
	if fsType == "ext4" || fsType == "ext3" {
		mkfsArgs = []string{
			"-F",  // Force flag
			"-m0", // Zero blocks reserved for privileged processes
			source,
		}
	}

	if !luksContext.EncryptionEnabled {
		m.log.WithFields(logrus.Fields{
			"cmd":  mkfsCmd,
			"args": mkfsArgs,
		}).Info("executing format command")

		out, err := exec.Command(mkfsCmd, mkfsArgs...).CombinedOutput()
		if err != nil {
			return fmt.Errorf("formatting disk failed: %v cmd: '%s %s' output: %q",
				err, mkfsCmd, strings.Join(mkfsArgs, " "), string(out))
		}

		return nil
	} else {
		err := luksContext.validate()
		if err != nil {
			return err
		}
		err = luksFormat(source, mkfsCmd, mkfsArgs, luksContext, m.log)
		if err != nil {
			return err
		}
		return nil
	}
}

func (m *mounter) Mount(source, target, fsType string, isBind bool, luksContext LuksContext, opts ...string) error {
	mountCmd := "mount"
	mountArgs := []string{}

	if fsType == "" {
		return errors.New("fs type is not specified for mounting the volume")
	}

	if source == "" {
		return errors.New("source is not specified for mounting the volume")
	}

	if target == "" {
		return errors.New("target is not specified for mounting the volume")
	}

	mountArgs = append(mountArgs, "-t", fsType)

	if len(opts) > 0 {
		mountArgs = append(mountArgs, "-o", strings.Join(opts, ","))
	}

	if !isBind && luksContext.EncryptionEnabled {
		luksSource, err := luksPrepareMount(source, luksContext, m.log)
		if err != nil {
			m.log.WithFields(logrus.Fields{
				"error":  err.Error(),
				"volume": luksContext.VolumeName,
			}).Error("failed to prepare luks volume for mounting")
			return err
		}
		mountArgs = append(mountArgs, luksSource)
	} else {
		mountArgs = append(mountArgs, source)
	}

	mountArgs = append(mountArgs, target)

	// create target, os.Mkdirall is noop if it exists
	err := os.MkdirAll(target, 0750)
	if err != nil {
		return err
	}

	m.log.WithFields(logrus.Fields{
		"cmd":  mountCmd,
		"args": mountArgs,
	}).Info("executing mount command")

	out, err := exec.Command(mountCmd, mountArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mounting failed: %v cmd: '%s %s' output: %q",
			err, mountCmd, strings.Join(mountArgs, " "), string(out))
	}

	return nil
}

func (m *mounter) Unmount(target string, isStaging bool) error {
	if target == "" {
		return errors.New("target is not specified for unmounting the volume")
	}

	// if this is the unmount call after the mount-bind has been removed,
	// a luks volume needs to be closed after unmounting; get the source
	// of the mount to check if that is a luks volume
	mountSources, err := getMountSources(target)
	if err != nil {
		return err
	}
	if len(mountSources) == 0 {
		return fmt.Errorf("unable to determine mount sources of target %s", target)
	}

	umountCmd := "umount"
	umountArgs := []string{target}

	m.log.WithFields(logrus.Fields{
		"cmd":  umountCmd,
		"args": umountArgs,
	}).Info("executing umount command")

	out, err := exec.Command(umountCmd, umountArgs...).CombinedOutput()
	if err != nil {
		return fmt.Errorf("unmounting failed: %v cmd: '%s %s' output: %q",
			err, umountCmd, target, string(out))
	}

	// if this us the unstaging process, check if the source is a luks volume and close it
	if isStaging {
		for _, source := range mountSources {
			isLuksMapping, mappingName, err := isLuksMapping(source)
			if err != nil {
				return err
			}
			if isLuksMapping {
				err := luksClose(mappingName, m.log)
				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// gets the mount sources of a mountpoint
func getMountSources(target string) ([]string, error) {
	_, err := exec.LookPath("findmnt")
	if err != nil {
		if err == exec.ErrNotFound {
			return nil, fmt.Errorf("%q executable not found in $PATH", "findmnt")
		}
		return nil, err
	}
	out, err := exec.Command("sh", "-c", fmt.Sprintf("findmnt -o SOURCE -n -M %s", target)).CombinedOutput()
	if err != nil {
		// findmnt exits with non zero exit status if it couldn't find anything
		if strings.TrimSpace(string(out)) == "" {
			return nil, nil
		}
		return nil, fmt.Errorf("checking mounted failed: %v cmd: %q output: %q",
			err, "findmnt", string(out))
	}
	return strings.Split(string(out), "\n"), nil
}

func (m *mounter) IsFormatted(source string, luksContext LuksContext) (bool, error) {
	if !luksContext.EncryptionEnabled {
		return isVolumeFormatted(source, m.log)
	}

	formatted, err := isLuksVolumeFormatted(source, luksContext, m.log)
	if err != nil {
		return false, err
	}
	return formatted, nil
}

func isVolumeFormatted(source string, log *logrus.Entry) (bool, error) {
	if source == "" {
		return false, errors.New("source is not specified")
	}

	blkidCmd := "blkid"
	_, err := exec.LookPath(blkidCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return false, fmt.Errorf("%q executable not found in $PATH", blkidCmd)
		}
		return false, err
	}

	blkidArgs := []string{source}

	log.WithFields(logrus.Fields{
		"cmd":  blkidCmd,
		"args": blkidArgs,
	}).Info("checking if source is formatted")

	out, err := exec.Command(blkidCmd, blkidArgs...).CombinedOutput()
	if err != nil {
		return false, fmt.Errorf("checking formatting failed: %v cmd: %q output: %q",
			err, blkidCmd, string(out))
	}

	if strings.TrimSpace(string(out)) == "" {
		return false, nil
	}

	return true, nil
}

func (m *mounter) IsMounted(target string) (bool, error) {
	if target == "" {
		return false, errors.New("target is not specified for checking the mount")
	}

	findmntCmd := "findmnt"
	_, err := exec.LookPath(findmntCmd)
	if err != nil {
		if err == exec.ErrNotFound {
			return false, fmt.Errorf("%q executable not found in $PATH", findmntCmd)
		}
		return false, err
	}

	findmntArgs := []string{"-o", "TARGET,PROPAGATION,FSTYPE,OPTIONS", "-M", target, "-J"}

	m.log.WithFields(logrus.Fields{
		"cmd":  findmntCmd,
		"args": findmntArgs,
	}).Info("checking if target is mounted")

	out, err := exec.Command(findmntCmd, findmntArgs...).CombinedOutput()
	if err != nil {
		// findmnt exits with non zero exit status if it couldn't find anything
		if strings.TrimSpace(string(out)) == "" {
			return false, nil
		}

		return false, fmt.Errorf("checking mounted failed: %v cmd: %q output: %q",
			err, findmntCmd, string(out))
	}

	// no response means there is no mount
	if string(out) == "" {
		return false, nil
	}

	var resp *findmntResponse
	err = json.Unmarshal(out, &resp)
	if err != nil {
		return false, fmt.Errorf("couldn't unmarshal data: %q: %s", string(out), err)
	}

	targetFound := false
	for _, fs := range resp.FileSystems {
		// check if the mount is propagated correctly. It should be set to shared.
		if fs.Propagation != "shared" {
			return true, fmt.Errorf("mount propagation for target %q is not enabled", target)
		}

		// the mountpoint should match as well
		if fs.Target == target {
			targetFound = true
		}
	}

	return targetFound, nil
}
