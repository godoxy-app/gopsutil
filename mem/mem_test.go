// SPDX-License-Identifier: BSD-3-Clause
package mem

import (
	"errors"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shirou/gopsutil/v4/internal/common"
)

func TestVirtualMemory(t *testing.T) {
	if runtime.GOOS == "solaris" || runtime.GOOS == "illumos" {
		t.Skip("Only .Total .Available are supported on Solaris/illumos")
	}

	v, err := VirtualMemory()
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoError(t, err)
	t.Log(v)

	assert.Positive(t, v.Total())
	assert.Positive(t, v.Available)
	assert.Positive(t, v.Used)

	total := v.Used + v.Available
	totalStr := "used + available"
	assert.Equalf(t, v.Total(), total,
		"Total should be computable (%v): %v", totalStr, v)

	inDelta := assert.InDelta
	if runtime.GOOS == "windows" {
		inDelta = assert.InEpsilon
	}
	inDelta(t, v.UsedPercent(),
		100*float64(v.Used)/float64(v.Total()), 0.1,
		"UsedPercent should be how many percent of Total is Used: %v", v)
}

func TestSwapMemory(t *testing.T) {
	v, err := SwapMemory()
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoError(t, err)
	empty := &SwapMemoryStat{}
	assert.NotSamef(t, v, empty, "error %v", v)

	t.Log(v)
}

func TestVirtualMemoryStat_String(t *testing.T) {
	v := VirtualMemoryStat{
		Available: 20,
		Used:      30,
	}
	t.Log(v)
	e := `{"available":20,"used":30,"total":50,"used_percent":60.0}`
	assert.JSONEqf(t, e, v.String(), "VirtualMemoryStat string is invalid: %v", v)
}

func TestSwapMemoryStat_String(t *testing.T) {
	v := SwapMemoryStat{
		Total:       10,
		Used:        30,
		Free:        40,
		UsedPercent: 30.1,
		Sin:         1,
		Sout:        2,
		PgIn:        3,
		PgOut:       4,
		PgFault:     5,
		PgMajFault:  6,
	}
	e := `{"total":10,"used":30,"free":40,"usedPercent":30.1,"sin":1,"sout":2,"pgIn":3,"pgOut":4,"pgFault":5,"pgMajFault":6}`
	assert.JSONEqf(t, e, v.String(), "SwapMemoryStat string is invalid: %v", v)
}

func TestSwapDevices(t *testing.T) {
	v, err := SwapDevices()
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoErrorf(t, err, "error calling SwapDevices: %v", err)

	t.Logf("SwapDevices() -> %+v", v)

	require.NotEmptyf(t, v, "no swap devices found. [this is expected if the host has swap disabled]")

	for _, device := range v {
		require.NotEmptyf(t, device.Name, "deviceName not set in %+v", device)
		if device.FreeBytes == 0 {
			t.Logf("[WARNING] free-bytes is zero in %+v. This might be expected", device)
		}
		if device.UsedBytes == 0 {
			t.Logf("[WARNING] used-bytes is zero in %+v. This might be expected", device)
		}
	}
}
