// SPDX-License-Identifier: BSD-3-Clause
//go:build linux

package mem

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"math"
	"os"
	"strconv"
	"strings"

	"golang.org/x/sys/unix"

	"github.com/shirou/gopsutil/v4/internal/common"
)

func VirtualMemory() (VirtualMemoryStat, error) {
	return VirtualMemoryWithContext(context.Background())
}

func VirtualMemoryWithContext(ctx context.Context) (VirtualMemoryStat, error) {
	vm, _, err := fillFromMeminfoWithContext(ctx)
	if err != nil {
		return VirtualMemoryStat{}, err
	}
	return vm, nil
}

func fillFromMeminfoWithContext(ctx context.Context) (ret VirtualMemoryStat, retEx ExVirtualMemory, err error) {
	filename := common.HostProcWithContext(ctx, "meminfo")
	lines, _, err := common.ReadLines(filename)
	if err != nil {
		return ret, retEx, fmt.Errorf("couldn't read %s: %w", filename, err)
	}

	// flag if MemAvailable is in /proc/meminfo (kernel 3.14+)
	memavail := false
	activeFile := false      // "Active(file)" not available: 2.6.28 / Dec 2008
	inactiveFile := false    // "Inactive(file)" not available: 2.6.28 / Dec 2008
	hasSreclaimable := false // "Sreclaimable:" not available: 2.6.19 / Nov 2006

	total := uint64(0)
	sReclaimable := uint64(0)
	cached := uint64(0)
	free := uint64(0)

	for line := range lines {
		if strings.Count(line, ":") != 1 {
			continue
		}
		key, value, _ := strings.Cut(line, ":")
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		value = strings.ReplaceAll(value, " kB", "")

		switch key {
		case "MemTotal":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return ret, retEx, err
			}
			total = t * 1024
		case "MemFree":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return ret, retEx, err
			}
			free = t * 1024
		case "MemAvailable":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return ret, retEx, err
			}
			memavail = true
			ret.Available = t * 1024
		case "Cached":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return ret, retEx, err
			}
			cached = t * 1024
		case "Active(file)":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return ret, retEx, err
			}
			activeFile = true
			retEx.ActiveFile = t * 1024
		case "Inactive(file)":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return ret, retEx, err
			}
			inactiveFile = true
			retEx.InactiveFile = t * 1024
		case "SReclaimable":
			t, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				return ret, retEx, err
			}
			hasSreclaimable = true
			sReclaimable = t * 1024
		}
	}

	cached += sReclaimable

	if !memavail {
		if activeFile && inactiveFile && hasSreclaimable {
			ret.Available = calculateAvailVmem(ctx, &retEx, free, cached, sReclaimable)
		} else {
			ret.Available = cached + free
		}
	}
	ret.Used = total - ret.Available
	return ret, retEx, nil
}

func SwapMemory() (*SwapMemoryStat, error) {
	return SwapMemoryWithContext(context.Background())
}

func SwapMemoryWithContext(ctx context.Context) (*SwapMemoryStat, error) {
	sysinfo := &unix.Sysinfo_t{}

	if err := unix.Sysinfo(sysinfo); err != nil {
		return nil, err
	}
	ret := SwapMemoryStat{
		Total: uint64(sysinfo.Totalswap) * uint64(sysinfo.Unit),
		Free:  uint64(sysinfo.Freeswap) * uint64(sysinfo.Unit),
	}
	ret.Used = ret.Total - ret.Free
	// check Infinity
	if ret.Total != 0 {
		ret.UsedPercent = float64(ret.Total-ret.Free) / float64(ret.Total) * 100.0
	} else {
		ret.UsedPercent = 0
	}
	filename := common.HostProcWithContext(ctx, "vmstat")
	lines, _, err := common.ReadLines(filename)
	if err != nil {
		return nil, fmt.Errorf("couldn't read %s: %w", filename, err)
	}
	for l := range lines {
		fields := strings.Fields(l)
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "pswpin":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			ret.Sin = value * 4 * 1024
		case "pswpout":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			ret.Sout = value * 4 * 1024
		case "pgpgin":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			ret.PgIn = value * 4 * 1024
		case "pgpgout":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			ret.PgOut = value * 4 * 1024
		case "pgfault":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			ret.PgFault = value * 4 * 1024
		case "pgmajfault":
			value, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				continue
			}
			ret.PgMajFault = value * 4 * 1024
		}
	}
	return &ret, nil
}

// calculateAvailVmem is a fallback under kernel 3.14 where /proc/meminfo does not provide
// "MemAvailable:" column. It reimplements an algorithm from the link below
// https://github.com/giampaolo/psutil/pull/890
func calculateAvailVmem(ctx context.Context, retEx *ExVirtualMemory, free, cached, sReclaimable uint64) uint64 {
	var watermarkLow uint64

	fn := common.HostProcWithContext(ctx, "zoneinfo")
	lines, _, err := common.ReadLines(fn)
	if err != nil {
		return free + cached // fallback under kernel 2.6.13
	}

	pagesize := uint64(os.Getpagesize())
	watermarkLow = 0

	for line := range lines {
		fields := strings.Fields(line)

		if strings.HasPrefix(fields[0], "low") {
			lowValue, err := strconv.ParseUint(fields[1], 10, 64)
			if err != nil {
				lowValue = 0
			}
			watermarkLow += lowValue
		}
	}

	watermarkLow *= pagesize

	availMemory := free - watermarkLow
	pageCache := retEx.ActiveFile + retEx.InactiveFile
	pageCache -= uint64(math.Min(float64(pageCache/2), float64(watermarkLow)))
	availMemory += pageCache
	availMemory += sReclaimable - uint64(math.Min(float64(sReclaimable/2.0), float64(watermarkLow)))

	return availMemory
}

const swapsFilename = "swaps"

// swaps file column indexes
const (
	nameCol = 0
	// typeCol     = 1
	totalCol = 2
	usedCol  = 3
	// priorityCol = 4
)

func SwapDevices() ([]*SwapDevice, error) {
	return SwapDevicesWithContext(context.Background())
}

func SwapDevicesWithContext(ctx context.Context) ([]*SwapDevice, error) {
	swapsFilePath := common.HostProcWithContext(ctx, swapsFilename)
	f, err := os.Open(swapsFilePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return parseSwapsFile(ctx, f)
}

func parseSwapsFile(ctx context.Context, r io.Reader) ([]*SwapDevice, error) {
	swapsFilePath := common.HostProcWithContext(ctx, swapsFilename)
	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("couldn't read file %q: %w", swapsFilePath, err)
		}
		return nil, fmt.Errorf("unexpected end-of-file in %q", swapsFilePath)

	}

	// Check header headerFields are as expected
	headerFields := strings.Fields(scanner.Text())
	if len(headerFields) <= usedCol {
		return nil, fmt.Errorf("couldn't parse %q: too few fields in header", swapsFilePath)
	}
	if headerFields[nameCol] != "Filename" {
		return nil, fmt.Errorf("couldn't parse %q: expected %q to be %q", swapsFilePath, headerFields[nameCol], "Filename")
	}
	if headerFields[totalCol] != "Size" {
		return nil, fmt.Errorf("couldn't parse %q: expected %q to be %q", swapsFilePath, headerFields[totalCol], "Size")
	}
	if headerFields[usedCol] != "Used" {
		return nil, fmt.Errorf("couldn't parse %q: expected %q to be %q", swapsFilePath, headerFields[usedCol], "Used")
	}

	var swapDevices []*SwapDevice
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) <= usedCol {
			return nil, fmt.Errorf("couldn't parse %q: too few fields", swapsFilePath)
		}

		totalKiB, err := strconv.ParseUint(fields[totalCol], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse 'Size' column in %q: %w", swapsFilePath, err)
		}

		usedKiB, err := strconv.ParseUint(fields[usedCol], 10, 64)
		if err != nil {
			return nil, fmt.Errorf("couldn't parse 'Used' column in %q: %w", swapsFilePath, err)
		}

		swapDevices = append(swapDevices, &SwapDevice{
			Name:      fields[nameCol],
			UsedBytes: usedKiB * 1024,
			FreeBytes: (totalKiB - usedKiB) * 1024,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("couldn't read file %q: %w", swapsFilePath, err)
	}

	return swapDevices, nil
}
