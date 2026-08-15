package disk

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExcludeDisks(t *testing.T) {
	tests := []struct {
		name          string
		shouldExclude bool
	}{
		{
			name:          "nvme0",
			shouldExclude: false,
		},
		{
			name:          "nvme0n1",
			shouldExclude: true,
		},
		{
			name:          "nvme0n1p1",
			shouldExclude: true,
		},
		{
			name:          "sda",
			shouldExclude: false,
		},
		{
			name:          "sda1",
			shouldExclude: true,
		},
		{
			name:          "hda",
			shouldExclude: false,
		},
		{
			name:          "vda",
			shouldExclude: false,
		},
		{
			name:          "xvda",
			shouldExclude: false,
		},
		{
			name:          "xva",
			shouldExclude: true,
		},
		{
			name:          "loop0",
			shouldExclude: true,
		},
		{
			name:          "mmcblk0",
			shouldExclude: false,
		},
		{
			name:          "mmcblk0p1",
			shouldExclude: true,
		},
		{
			name:          "ab",
			shouldExclude: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldExcludeDisk(tt.name)
			assert.Equal(t, tt.shouldExclude, result)
		})
	}
}
