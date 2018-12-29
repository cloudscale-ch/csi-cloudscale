package driver

import (
	"testing"
	"github.com/container-storage-interface/spec/lib/go/csi"
)

func TestCalculateStorageGBEmpty(t *testing.T) {
	value, err := calculateStorageGB(nil)
	if value != 1 {
		t.Fatalf("Expected 1, not %v", value)
	}
	if err != nil {
		t.Fatal("Expected no error")
	}
}

func TestCalculateStorageGBLimitTooLow(t *testing.T) {
	_, err := calculateStorageGB(&csi.CapacityRange {LimitBytes: 1})
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestCalculateStorageGBNotPossible(t *testing.T) {
	base := int64(50 * GB)
	_, err := calculateStorageGB(&csi.CapacityRange {RequiredBytes: base + 1, LimitBytes: base + 2})
	if err == nil {
		t.Fatal("Expected an error")
	}
}

func TestCalculateStorageGBEdges(t *testing.T) {
	base := int64(50 * GB)
	value, err := calculateStorageGB(&csi.CapacityRange {RequiredBytes: base, LimitBytes: base * 2})
	if err != nil {
		t.Fatal("Expected no error")
	}
	if value != 50 {
		t.Fatalf("Expected 50, not %v", value)
	}
}

func TestCalculateStorageGBRounding(t *testing.T) {
	base := int64(30 * GB)
	value, err := calculateStorageGB(&csi.CapacityRange {RequiredBytes: base + 1})
	if err != nil {
		t.Fatal("Expected no error")
	}
	if value != 31 {
		t.Fatalf("Expected 31, not %v", value)
	}

	value, err = calculateStorageGB(&csi.CapacityRange {RequiredBytes: base - 1})
	if err != nil {
		t.Fatal("Expected no error")
	}
	if value != 30 {
		t.Fatalf("Expected 30, not %v", value)
	}
}
