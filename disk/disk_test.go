// SPDX-License-Identifier: BSD-3-Clause
package disk

import (
	"errors"
	"runtime"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shirou/gopsutil/v4/internal/common"
)

func TestUsage(t *testing.T) {
	path := "/"
	if runtime.GOOS == "windows" {
		path = "C:"
	}
	v, err := Usage(path)
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}

	require.NoError(t, err)
	assert.Equalf(t, v.Path, path, "error %v", err)
}

func TestPartitions(t *testing.T) {
	ret, err := Partitions(false)
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}

	if err != nil || len(ret) == 0 {
		t.Errorf("error %v", err)
	}
	t.Log(ret)

	assert.NotEmptyf(t, ret, "ret is empty")
	for _, disk := range ret {
		assert.NotEmptyf(t, disk.Device, "Could not get device info %v", disk)
	}
}

func TestIOCounters(t *testing.T) {
	ret, err := IOCounters()
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}

	require.NoError(t, err)
	assert.NotEmptyf(t, ret, "ret is empty")
	empty := IOCountersStat{}
	for part, io := range ret {
		t.Log(part, io)
		assert.NotEqualf(t, io, empty, "io_counter error %v, %v", part, io)
	}
}

// https://github.com/shirou/gopsutil/issues/560 regression test
func TestIOCounters_concurrency_on_darwin_cgo(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("darwin only")
	}
	var wg sync.WaitGroup
	const maxCount = 1000
	for i := 1; i < maxCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			IOCounters()
		}()
	}
	wg.Wait()
}

func TestUsageStat_String(t *testing.T) {
	v := UsageStat{
		Path:   "/",
		Fstype: "ext4",
		Free:   2000,
		Used:   3000,
	}
	e := `{"path":"/","fstype":"ext4","total":5000,"free":2000,"used":3000,"used_percent":60.0}`
	assert.JSONEqf(t, e, v.String(), "DiskUsageStat string is invalid: %v", v)
}

func TestPartitionStat_String(t *testing.T) {
	v := PartitionStat{
		Device:     "sd01",
		Mountpoint: "/",
		Fstype:     "ext4",
	}
	e := `{"device":"sd01","mountpoint":"/","fstype":"ext4"}`
	assert.JSONEqf(t, e, v.String(), "DiskUsageStat string is invalid: %v", v)
}

func TestIOCountersStat_String(t *testing.T) {
	v := IOCountersStat{
		Name:       "sd01",
		ReadCount:  100,
		WriteCount: 200,
		ReadBytes:  300,
		WriteBytes: 400,
	}
	e := `{"name":"sd01","readBytes":300,"writeBytes":400,"readCount":100,"writeCount":200,"iops":0,"readSpeed":0.0,"writeSpeed":0.0}`
	assert.JSONEqf(t, e, v.String(), "DiskUsageStat string is invalid: %v", v)
}
