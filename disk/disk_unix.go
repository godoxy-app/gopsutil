// SPDX-License-Identifier: BSD-3-Clause
//go:build freebsd || linux || darwin

package disk

import (
	"context"
	"strconv"

	"github.com/shirou/gopsutil/v4/internal/common"
	"golang.org/x/sys/unix"
)

func UsageWithContext(_ context.Context, path string) (UsageStat, error) {
	stat := unix.Statfs_t{}
	err := unix.Statfs(path, &stat)
	if err != nil {
		return UsageStat{}, err
	}
	bsize := stat.Bsize

	ret := UsageStat{
		Path:   common.InternString(unescapeFstab(path)),
		Fstype: common.InternString(getFsType(stat)),
		Free:   (uint64(stat.Bavail) * uint64(bsize)),
	}

	ret.Used = (uint64(stat.Blocks) - uint64(stat.Bfree)) * uint64(bsize)

	return ret, nil
}

// Unescape escaped octal chars (like space 040, ampersand 046 and backslash 134) to their real value in fstab fields issue#555
func unescapeFstab(path string) string {
	escaped, err := strconv.Unquote(`"` + path + `"`)
	if err != nil {
		return path
	}
	return escaped
}
