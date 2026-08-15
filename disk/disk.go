// SPDX-License-Identifier: BSD-3-Clause
package disk

import (
	"context"
	"encoding/json"

	"github.com/shirou/gopsutil/v4/internal/common"
)

var invoke common.Invoker = common.Invoke{}

type (
	Warnings = common.Warnings
	u64p     = common.PlaceHolder[uint64]
	f64p     = common.PlaceHolder[float64]
)

type UsageStat struct {
	Path              string  `json:"path"`
	Fstype            string  `json:"fstype"`
	Total_            u64p    `json:"total" swaggertype:"number"`
	Free              uint64  `json:"free"`
	Used              uint64  `json:"used"`
	UsedPercent_      f64p    `json:"used_percent" swaggertype:"number"`
	InodesUsedPercent float64 `json:"inodesUsedPercent"`
}

type usageStatJSON struct {
	Path        string  `json:"path"`
	Fstype      string  `json:"fstype"`
	Total       uint64  `json:"total"`
	Free        uint64  `json:"free"`
	Used        uint64  `json:"used"`
	UsedPercent float64 `json:"used_percent"`
}

func (d *UsageStat) Total() uint64 {
	return d.Free + d.Used
}

func (d *UsageStat) UsedPercent() float64 {
	total := d.Total()
	if total == 0 {
		return 0
	}
	return float64(d.Used) / float64(total) * 100
}

func (d UsageStat) MarshalJSON() ([]byte, error) {
	return json.Marshal(usageStatJSON{
		Path:        d.Path,
		Fstype:      d.Fstype,
		Total:       d.Total(),
		Free:        d.Free,
		Used:        d.Used,
		UsedPercent: d.UsedPercent(),
	})
}

type PartitionStat struct {
	Device     string `json:"device"`
	Mountpoint string `json:"mountpoint"`
	Fstype     string `json:"fstype"`
}

type IOCountersStat struct {
	ReadCount  uint64 `json:"readCount"`
	WriteCount uint64 `json:"writeCount"`
	ReadBytes  uint64 `json:"readBytes"`
	WriteBytes uint64 `json:"writeBytes"`

	Name string `json:"name"`

	IOCountersStatExtra
}

// godoxy
type IOCountersStatExtra struct {
	Iops       uint64  `json:"iops"`
	ReadSpeed  float32 `json:"readSpeed"`
	WriteSpeed float32 `json:"writeSpeed"`
}

func (d UsageStat) String() string {
	s, _ := d.MarshalJSON()
	return string(s)
}

func (d PartitionStat) String() string {
	s, _ := json.Marshal(d)
	return string(s)
}

func (d IOCountersStat) String() string {
	s, _ := json.Marshal(d)
	return string(s)
}

// Usage returns a file system usage. path is a filesystem path such
// as "/", not device file path like "/dev/vda1".  If you want to use
// a return value of disk.Partitions, use "Mountpoint" not "Device".
func Usage(path string) (UsageStat, error) {
	return UsageWithContext(context.Background(), path)
}

// Partitions returns disk partitions. If all is false, returns
// physical devices only (e.g. hard disks, cd-rom drives, USB keys)
// and ignore all others (e.g. memory partitions such as /dev/shm)
//
// 'all' argument is ignored for BSD, see: https://github.com/giampaolo/psutil/issues/906
func Partitions(all bool) ([]PartitionStat, error) {
	return PartitionsWithContext(context.Background(), all)
}

func IOCounters(names ...string) (map[string]*IOCountersStat, error) {
	return IOCountersWithContext(context.Background(), names...)
}

// SerialNumber returns Serial Number of given device or empty string
// on error. Name of device is expected, eg. /dev/sda
func SerialNumber(name string) (string, error) {
	return SerialNumberWithContext(context.Background(), name)
}

// Label returns label of given device or empty string on error.
// Name of device is expected, eg. /dev/sda
// Supports label based on devicemapper name
// See https://www.kernel.org/doc/Documentation/ABI/testing/sysfs-block-dm
func Label(name string) (string, error) {
	return LabelWithContext(context.Background(), name)
}
