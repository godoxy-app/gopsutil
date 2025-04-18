// SPDX-License-Identifier: BSD-3-Clause
//go:build linux

package mem

import (
	"encoding/json"
)

type ExVirtualMemory struct {
	ActiveFile   uint64 `json:"activefile"`
	InactiveFile uint64 `json:"inactivefile"`
}

func (v ExVirtualMemory) String() string {
	s, _ := json.Marshal(v)
	return string(s)
}

type ExLinux struct{}

func NewExLinux() *ExLinux {
	return &ExLinux{}
}
