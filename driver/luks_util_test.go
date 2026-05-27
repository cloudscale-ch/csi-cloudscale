/*
Copyright cloudscale.ch

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0
*/

package driver

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseCryptsetupStatus(t *testing.T) {
	tests := []struct {
		name           string
		out            string
		wantBacking    string
		wantIsLuks     bool
		wantIsInactive bool
	}{
		{
			name: "active LUKS1 mapping",
			out: `/dev/mapper/pvc-foo is active.
  type:  LUKS1
  cipher:  aes-xts-plain64
  keysize: 512 bits
  device:  /dev/sdb
  offset:  4096 sectors
  size:    1048576 sectors
  mode:    read/write`,
			wantBacking: "/dev/sdb",
			wantIsLuks:  true,
		},
		{
			name: "active non-LUKS mapping",
			out: `/dev/mapper/foo is active.
  type:  PLAIN
  cipher:  aes-cbc-essiv:sha256
  device:  /dev/sdc`,
			wantBacking: "/dev/sdc",
			wantIsLuks:  false,
		},
		{
			name:           "inactive mapping",
			out:            `/dev/mapper/pvc-foo is inactive.`,
			wantIsInactive: true,
		},
		{
			name: "malformed (no type or device line)",
			out:  `/dev/mapper/pvc-foo is active.`,
		},
		{
			name:           "empty output",
			out:            ``,
			wantIsInactive: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseCryptsetupStatus([]byte(tt.out))
			if got.backing != tt.wantBacking {
				t.Errorf("backing = %q, want %q", got.backing, tt.wantBacking)
			}
			if got.isLuks != tt.wantIsLuks {
				t.Errorf("isLuks = %v, want %v", got.isLuks, tt.wantIsLuks)
			}
			if got.isInactive != tt.wantIsInactive {
				t.Errorf("isInactive = %v, want %v", got.isInactive, tt.wantIsInactive)
			}
		})
	}
}

func TestValidateExistingLuksMapping(t *testing.T) {
	// EvalSymlinks on a regular file returns the file's absolute path. That is
	// enough to exercise the match / mismatch branches without needing a real
	// block device or /dev/mapper entry.
	tmp, err := os.CreateTemp(t.TempDir(), "fake-backing-*")
	if err != nil {
		t.Fatalf("CreateTemp: %v", err)
	}
	_ = tmp.Close()
	resolved, err := filepath.EvalSymlinks(tmp.Name())
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}

	tests := []struct {
		name         string
		status       cryptsetupStatusInfo
		statusErr    error
		volume       string
		wantInactive bool
		wantBacking  string
		wantErrSub   string
	}{
		{
			name:        "backing matches",
			status:      cryptsetupStatusInfo{backing: resolved},
			volume:      tmp.Name(),
			wantBacking: resolved,
		},
		{
			name:       "backing mismatches",
			status:     cryptsetupStatusInfo{backing: "/dev/nonexistent-other"},
			volume:     tmp.Name(),
			wantErrSub: "refusing to reuse stale mapping",
		},
		{
			name:         "mapping inactive",
			status:       cryptsetupStatusInfo{isInactive: true},
			volume:       tmp.Name(),
			wantInactive: true,
		},
		{
			name:       "no backing device reported",
			status:     cryptsetupStatusInfo{},
			volume:     tmp.Name(),
			wantErrSub: "reported no backing device",
		},
		{
			name:       "status call errors",
			statusErr:  errors.New("boom"),
			volume:     tmp.Name(),
			wantErrSub: "cryptsetup status failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fake := func(string) (cryptsetupStatusInfo, error) {
				return tt.status, tt.statusErr
			}
			inactive, backing, err := validateExistingLuksMapping("mapper", tt.volume, fake)

			if tt.wantErrSub != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.wantErrSub)
				}
				if !strings.Contains(err.Error(), tt.wantErrSub) {
					t.Fatalf("expected error containing %q, got %q", tt.wantErrSub, err.Error())
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if inactive != tt.wantInactive {
				t.Errorf("inactive = %v, want %v", inactive, tt.wantInactive)
			}
			if backing != tt.wantBacking {
				t.Errorf("backing = %q, want %q", backing, tt.wantBacking)
			}
		})
	}
}
