// SPDX-License-Identifier: BSD-3-Clause
package mem

import (
	"encoding/json"

	"github.com/shirou/gopsutil/v4/internal/common"
)

var invoke common.Invoker = common.Invoke{}

// Memory usage statistics. Total, Available and Used contain numbers of bytes
// for human consumption.
//
// The other fields in this struct contain kernel specific values.
type VirtualMemoryStat struct {
	// Total amount of RAM on this system
	Total_ common.PlaceHolder[uint64] `json:"total" swaggertype:"number"`

	// RAM available for programs to allocate
	//
	// This value is computed from the kernel specific values.
	Available uint64 `json:"available"`

	// RAM used by programs
	//
	// This value is computed from the kernel specific values.
	Used uint64 `json:"used"`

	// Percentage of RAM used by programs
	//
	// This value is computed from the kernel specific values.
	UsedPercent_ common.PlaceHolder[float64] `json:"used_percent" swaggertype:"number"`
}

type virtualMemoryStatFull struct {
	Available   uint64  `json:"available"`
	Used        uint64  `json:"used"`
	Total       uint64  `json:"total"`
	UsedPercent float64 `json:"used_percent"`
}

func (m VirtualMemoryStat) Total() uint64 {
	return m.Available + m.Used
}

func (m VirtualMemoryStat) UsedPercent() float64 {
	total := m.Total()
	if total == 0 {
		return 0
	}
	return float64(m.Used) / float64(total) * 100
}

func (m VirtualMemoryStat) MarshalJSON() ([]byte, error) {
	return json.Marshal(virtualMemoryStatFull{
		Available:   m.Available,
		Used:        m.Used,
		Total:       m.Total(),
		UsedPercent: m.UsedPercent(),
	})
}

type SwapMemoryStat struct {
	Total       uint64  `json:"total"`
	Used        uint64  `json:"used"`
	Free        uint64  `json:"free"`
	UsedPercent float64 `json:"usedPercent"`
	Sin         uint64  `json:"sin"`
	Sout        uint64  `json:"sout"`
	PgIn        uint64  `json:"pgIn"`
	PgOut       uint64  `json:"pgOut"`
	PgFault     uint64  `json:"pgFault"`

	// Linux specific numbers
	// https://www.kernel.org/doc/Documentation/cgroup-v2.txt
	PgMajFault uint64 `json:"pgMajFault"`
}

func (m VirtualMemoryStat) String() string {
	s, _ := m.MarshalJSON()
	return string(s)
}

func (m SwapMemoryStat) String() string {
	s, _ := json.Marshal(m)
	return string(s)
}

type SwapDevice struct {
	Name      string `json:"name"`
	UsedBytes uint64 `json:"usedBytes"`
	FreeBytes uint64 `json:"freeBytes"`
}

func (m SwapDevice) String() string {
	s, _ := json.Marshal(m)
	return string(s)
}
