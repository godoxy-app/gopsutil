// SPDX-License-Identifier: BSD-3-Clause
//go:build linux

package mem

import (
	"context"
	"errors"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shirou/gopsutil/v4/internal/common"
)

var virtualMemoryTests = []struct {
	mockedRootFS string
	stat         VirtualMemoryStat
	exStat       ExVirtualMemory
}{
	{
		"intelcorei5",
		VirtualMemoryStat{
			Available: 11495358464,
			Used:      5006942208,
		},
		ExVirtualMemory{
			ActiveFile:   1121992 * 1024,
			InactiveFile: 1683344 * 1024,
		},
	},
	{
		"issue1002",
		VirtualMemoryStat{
			Available: 215199744,
			Used:      45379584,
		},
		ExVirtualMemory{
			ActiveFile:   88280 * 1024,
			InactiveFile: 8380 * 1024,
		},
	},
	{
		"anonhugepages",
		VirtualMemoryStat{
			Available: 127880216 * 1024,
			Used:      136109264896,
		},
		ExVirtualMemory{
			ActiveFile:   0,
			InactiveFile: 0,
		},
	},
}

func TestVirtualMemoryLinux(t *testing.T) {
	for _, tt := range virtualMemoryTests {
		t.Run(tt.mockedRootFS, func(t *testing.T) {
			t.Setenv("HOST_PROC", filepath.Join("testdata", "linux", "virtualmemory", tt.mockedRootFS, "proc"))

			stat, err := VirtualMemory()
			if errors.Is(err, common.ErrNotImplementedError) {
				t.Skip("not implemented")
			}
			require.NoError(t, err)
			assert.Truef(t, reflect.DeepEqual(stat, tt.stat), "got: %+v\nwant: %+v", stat, tt.stat)
		})
	}
}

func TestExVirtualMemoryLinux(t *testing.T) {
	for _, tt := range virtualMemoryTests {
		t.Run(tt.mockedRootFS, func(t *testing.T) {
			t.Setenv("HOST_PROC", filepath.Join("testdata", "linux", "virtualmemory", tt.mockedRootFS, "proc"))

			ex := NewExLinux()
			exStat, err := ex.VirtualMemory()
			if errors.Is(err, common.ErrNotImplementedError) {
				t.Skip("not implemented")
			}
			require.NoError(t, err)
			assert.Truef(t, reflect.DeepEqual(exStat, tt.exStat), "got: %+v\nwant: %+v", exStat, tt.exStat)
		})
	}
}

const validFile = `Filename				Type		Size		Used		Priority
/dev/dm-2                               partition	67022844	490788		-2
/swapfile                               file		2		1		-3
`

const invalidFile = `INVALID				Type		Size		Used		Priority
/dev/dm-2                               partition	67022844	490788		-2
/swapfile                               file		1048572		0		-3
`

func TestParseSwapsFile_ValidFile(t *testing.T) {
	stats, err := parseSwapsFile(context.Background(), strings.NewReader(validFile))
	require.NoError(t, err)

	assert.Equal(t, SwapDevice{
		Name:      "/dev/dm-2",
		UsedBytes: 502566912,
		FreeBytes: 68128825344,
	}, *stats[0])

	assert.Equal(t, SwapDevice{
		Name:      "/swapfile",
		UsedBytes: 1024,
		FreeBytes: 1024,
	}, *stats[1])
}

func TestParseSwapsFile_InvalidFile(t *testing.T) {
	_, err := parseSwapsFile(context.Background(), strings.NewReader(invalidFile))
	assert.Error(t, err)
}

func TestParseSwapsFile_EmptyFile(t *testing.T) {
	_, err := parseSwapsFile(context.Background(), strings.NewReader(""))
	assert.Error(t, err)
}

// A data row with too few columns must return an error, not panic.
func TestParseSwapsFile_ShortDataRow(t *testing.T) {
	shortRow := "Filename\tType\tSize\tUsed\tPriority\n/dev/dm-2\tpartition\t1024\n"
	_, err := parseSwapsFile(context.Background(), strings.NewReader(shortRow))
	assert.Error(t, err)
}

// A header row with too few columns must return an error, not panic.
func TestParseSwapsFile_ShortHeader(t *testing.T) {
	shortHeader := "Filename\tType\tSize\n"
	_, err := parseSwapsFile(context.Background(), strings.NewReader(shortHeader))
	assert.Error(t, err)
}
