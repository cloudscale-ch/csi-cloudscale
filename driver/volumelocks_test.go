/*
Copyright 2019 The Kubernetes Authors.

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
	"sync"
	"testing"
	"time"
)

func TestVolumeLocks_TryAcquire_Success(t *testing.T) {
	vl := NewVolumeLocks()

	volumeID := "test-volume-1"
	if acquired := vl.TryAcquire(volumeID); !acquired {
		t.Errorf("TryAcquire should succeed on first call, got false")
	}
}

func TestVolumeLocks_TryAcquire_AlreadyLocked(t *testing.T) {
	vl := NewVolumeLocks()

	volumeID := "test-volume-1"

	// First acquire should succeed
	if acquired := vl.TryAcquire(volumeID); !acquired {
		t.Fatalf("First TryAcquire should succeed")
	}

	// Second acquire on same volume should fail
	if acquired := vl.TryAcquire(volumeID); acquired {
		t.Errorf("TryAcquire should fail when volume is already locked, got true")
	}
}

func TestVolumeLocks_Release(t *testing.T) {
	vl := NewVolumeLocks()

	volumeID := "test-volume-1"

	// Acquire the lock
	if acquired := vl.TryAcquire(volumeID); !acquired {
		t.Fatalf("First TryAcquire should succeed")
	}

	// Release the lock
	vl.Release(volumeID)

	// Should be able to acquire again after release
	if acquired := vl.TryAcquire(volumeID); !acquired {
		t.Errorf("TryAcquire should succeed after Release, got false")
	}
}

func TestVolumeLocks_Release_NonExistent(t *testing.T) {
	vl := NewVolumeLocks()

	// Releasing a non-existent lock should not panic
	vl.Release("non-existent-volume")
}

func TestVolumeLocks_MultipleVolumes(t *testing.T) {
	vl := NewVolumeLocks()

	volume1 := "test-volume-1"
	volume2 := "test-volume-2"

	// Acquire lock on volume1
	if acquired := vl.TryAcquire(volume1); !acquired {
		t.Fatalf("TryAcquire for volume1 should succeed")
	}

	// Should still be able to acquire lock on volume2
	if acquired := vl.TryAcquire(volume2); !acquired {
		t.Errorf("TryAcquire for volume2 should succeed when volume1 is locked")
	}

	// volume1 should still be locked
	if acquired := vl.TryAcquire(volume1); acquired {
		t.Errorf("TryAcquire for volume1 should fail when already locked")
	}
}

func TestVolumeLocks_ConcurrentAcquire(t *testing.T) {
	vl := NewVolumeLocks()
	volumeID := "test-volume-1"

	const numGoroutines = 100
	successCount := 0
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Start many goroutines trying to acquire the same lock
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if acquired := vl.TryAcquire(volumeID); acquired {
				mu.Lock()
				successCount++
				mu.Unlock()
				// Hold the lock briefly
				time.Sleep(time.Millisecond)
				vl.Release(volumeID)
			}
		}()
	}

	wg.Wait()

	// At least one goroutine should have succeeded
	if successCount == 0 {
		t.Error("At least one goroutine should have acquired the lock")
	}

	// The lock should be available now
	if acquired := vl.TryAcquire(volumeID); !acquired {
		t.Error("Lock should be available after all goroutines complete")
	}
}

func TestVolumeLocks_ConcurrentDifferentVolumes(t *testing.T) {
	vl := NewVolumeLocks()

	const numVolumes = 50
	var wg sync.WaitGroup
	results := make([]bool, numVolumes)

	// Each goroutine tries to lock a different volume
	for i := 0; i < numVolumes; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			volumeID := "test-volume-" + string(rune('A'+idx))
			results[idx] = vl.TryAcquire(volumeID)
		}(i)
	}

	wg.Wait()

	// All should have succeeded since they're different volumes
	for i, result := range results {
		if !result {
			t.Errorf("Volume %d should have been acquired successfully", i)
		}
	}
}

func TestVolumeLocks_RapidAcquireRelease(t *testing.T) {
	vl := NewVolumeLocks()
	volumeID := "test-volume-1"

	const iterations = 1000

	for i := 0; i < iterations; i++ {
		if acquired := vl.TryAcquire(volumeID); !acquired {
			t.Fatalf("Iteration %d: TryAcquire should succeed after release", i)
		}
		vl.Release(volumeID)
	}
}

func TestVolumeLocks_ConcurrentAcquireReleaseDifferentVolumes(t *testing.T) {
	vl := NewVolumeLocks()

	const numGoroutines = 10
	const iterations = 100
	var wg sync.WaitGroup

	// Multiple goroutines doing acquire/release on different volumes
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(volumeNum int) {
			defer wg.Done()
			volumeID := "test-volume-" + string(rune('A'+volumeNum))
			for j := 0; j < iterations; j++ {
				if acquired := vl.TryAcquire(volumeID); acquired {
					// Simulate some work
					time.Sleep(time.Microsecond)
					vl.Release(volumeID)
				}
			}
		}(i)
	}

	wg.Wait()
}
