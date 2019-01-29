package driver

import (
	"github.com/container-storage-interface/spec/lib/go/csi"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestCalculateStorageGBEmpty(t *testing.T) {
	value, err := calculateStorageGB(nil, "")
	assert.Equal(t, 1, value)
	assert.NoError(t, err)
}

func TestCalculateStorageGBLimitTooLow(t *testing.T) {
	_, err := calculateStorageGB(&csi.CapacityRange{LimitBytes: 1}, "")
	assert.Error(t, err)
}

func TestCalculateStorageGBNotPossible(t *testing.T) {
	base := int64(50 * GB)
	_, err := calculateStorageGB(&csi.CapacityRange{RequiredBytes: base + 1, LimitBytes: base + 2}, "")
	assert.Error(t, err)
}

func TestCalculateStorageGBEdges(t *testing.T) {
	base := int64(50 * GB)
	value, err := calculateStorageGB(&csi.CapacityRange{RequiredBytes: base, LimitBytes: base * 2}, "")
	assert.NoError(t, err)
	assert.Equal(t, 50, value)
}

func TestCalculateStorageGBRounding(t *testing.T) {
	base := int64(30 * GB)
	value, err := calculateStorageGB(&csi.CapacityRange{RequiredBytes: base + 1}, "")
	assert.NoError(t, err)
	assert.Equal(t, 31, value)

	value, err = calculateStorageGB(&csi.CapacityRange{RequiredBytes: base - 1}, "")
	assert.NoError(t, err)
	assert.Equal(t, 30, value)
}

func TestRequiredBulkStorageSize(t *testing.T) {
	val, err := calcStorageGbBulk(100, 0)
	assert.NoError(t, err)
	assert.Equal(t, 100, val)
}

func TestDefaultToMinimumBulkStorageSize(t *testing.T) {
	// should default to the minimum storage size
	val, err := calcStorageGbBulk(0, 0)
	assert.NoError(t, err)
	assert.Equal(t, 100, val)
}

func TestLimitMustNotBeSmallerThanMinimumBulkStorageSize(t *testing.T) {
	// limit is smaller than minimum bulk disk size
	_, err := calcStorageGbBulk(5, 5)
	assert.Error(t, err)
}
func TestRequestedBytesSmallerThanMinimumSizeUsesMinimumBulkStorageSize(t *testing.T) {
	// use minimum bulk disk size if the limit allows it
	val, err := calcStorageGbBulk(5, 100)
	assert.NoError(t, err)
	assert.Equal(t, 100, val)
}

func calcStorageGbBulk(reqGb int, limitGb int) (int, error) {
	if reqGb == -1 {
		if limitGb == -1 {
			return calculateStorageGB(&csi.CapacityRange{
			}, "bulk")
		} else {
			return calculateStorageGB(&csi.CapacityRange{
				LimitBytes: int64(limitGb * GB),
			}, "bulk")
		}
	} else {
		if limitGb == -1 {
			return calculateStorageGB(&csi.CapacityRange{
				RequiredBytes: int64(reqGb * GB),
			}, "bulk")
		} else {
			return calculateStorageGB(&csi.CapacityRange{
				RequiredBytes: int64(reqGb * GB),
				LimitBytes:    int64(limitGb * GB),
			}, "bulk")
		}
	}
}
