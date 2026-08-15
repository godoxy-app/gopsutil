// SPDX-License-Identifier: BSD-3-Clause
//go:build linux

package common

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"
)

// cachedBootTime must be accessed via atomic.Load/StoreUint64
var cachedBootTime uint64

func NumProcs() (uint64, error) {
	return NumProcsWithContext(context.Background())
}

func NumProcsWithContext(ctx context.Context) (uint64, error) {
	f, err := os.Open(HostProcWithContext(ctx))
	if err != nil {
		return 0, err
	}
	defer f.Close()

	list, err := f.Readdirnames(-1)
	if err != nil {
		return 0, err
	}
	var cnt uint64

	for _, v := range list {
		if _, err = strconv.ParseUint(v, 10, 64); err == nil {
			cnt++
		}
	}

	return cnt, nil
}

func BootTimeWithContext(ctx context.Context, enableCache bool) (uint64, error) {
	if enableCache {
		t := atomic.LoadUint64(&cachedBootTime)
		if t != 0 {
			return t, nil
		}
	}

	system, role, err := VirtualizationWithContext(ctx)
	if err != nil {
		return 0, err
	}

	useStatFile := true
	if system == "lxc" && role == "guest" {
		// if lxc, /proc/uptime is used.
		useStatFile = false
	} else if system == "docker" && role == "guest" {
		// also docker, guest
		useStatFile = false
	}

	if useStatFile {
		t, err := readBootTimeStat(ctx)
		if err != nil {
			return 0, err
		}
		if enableCache {
			atomic.StoreUint64(&cachedBootTime, t)
		}

		return t, nil
	}

	filename := HostProcWithContext(ctx, "uptime")
	content, ok, err := ReadFileExpectOneLine(filename)
	if err != nil {
		return handleBootTimeFileReadErr(err)
	}
	currentTime := float64(time.Now().UnixNano()) / float64(time.Second)

	if !ok {
		return 0, errors.New("wrong uptime format")
	}
	f := strings.Fields(content)
	b, err := strconv.ParseFloat(f[0], 64)
	if err != nil {
		return 0, err
	}
	t := currentTime - b

	if enableCache {
		atomic.StoreUint64(&cachedBootTime, uint64(t))
	}

	return uint64(t), nil
}

func handleBootTimeFileReadErr(err error) (uint64, error) {
	if !os.IsPermission(err) {
		return 0, err
	}
	var info syscall.Sysinfo_t
	err = syscall.Sysinfo(&info)
	if err != nil {
		return 0, err
	}

	currentTime := time.Now().UnixNano() / int64(time.Second)
	t := currentTime - int64(info.Uptime)
	return uint64(t), nil
}

func readBootTimeStat(ctx context.Context) (uint64, error) {
	filename := HostProcWithContext(ctx, "stat")
	line, err := ReadLine(filename, "btime")
	if err != nil {
		return handleBootTimeFileReadErr(err)
	}
	if strings.HasPrefix(line, "btime") {
		f := strings.Fields(line)
		if len(f) != 2 {
			return 0, errors.New("wrong btime format")
		}
		b, err := strconv.ParseInt(f[1], 10, 64)
		if err != nil {
			return 0, err
		}
		t := uint64(b)
		return t, nil
	}
	return 0, errors.New("could not find btime")
}

func Virtualization() (string, string, error) {
	return VirtualizationWithContext(context.Background())
}

// required variables for concurrency safe virtualization caching
var (
	cachedVirtMap   map[string]string
	cachedVirtMutex sync.RWMutex
	cachedVirtOnce  sync.Once
)

func VirtualizationWithContext(ctx context.Context) (string, string, error) {
	var system, role string

	// if cached already, return from cache
	cachedVirtMutex.RLock() // unlock won't be deferred so concurrent reads don't wait for long
	if cachedVirtMap != nil {
		cachedSystem, cachedRole := cachedVirtMap["system"], cachedVirtMap["role"]
		cachedVirtMutex.RUnlock()
		return cachedSystem, cachedRole, nil
	}
	cachedVirtMutex.RUnlock()

	filename := HostProcWithContext(ctx, "xen")
	if PathExists(filename) {
		system = "xen"
		role = "guest" // assume guest

		if PathExists(filepath.Join(filename, "capabilities")) {
			contents, err := FileContains(filepath.Join(filename, "capabilities"), "control_d")
			if err == nil && contents {
				role = "host"
			}
		}
	}

	filename = HostProcWithContext(ctx, "modules")
	if PathExists(filename) {
		ok, match, _ := FileContainsAny(filename, "kvm,hv_util,vboxdrv,vboxguest,vmware")
		if ok {
			switch match {
			case "kvm":
				system = "kvm"
				role = "host"
			case "hv_util":
				system = "hyperv"
				role = "guest"
			case "vboxdrv":
				system = "vbox"
				role = "host"
			case "vboxguest":
				system = "vbox"
				role = "guest"
			case "vmware":
				system = "vmware"
				role = "guest"
			}
		}
	}

	filename = HostProcWithContext(ctx, "cpuinfo")
	if PathExists(filename) {
		ok, _, _ := FileContainsAny(filename, "QEMU Virtual CPU,Common KVM processor,Common 32-bit KVM processor")
		if ok {
			system = "kvm"
			role = "guest"
		}
	}

	filename = HostProcWithContext(ctx, "bus/pci/devices")
	if PathExists(filename) {
		ok, _ := FileContains(filename, "virtio-pci")
		if ok {
			role = "guest"
		}
	}

	filename = HostProcWithContext(ctx)
	if PathExists(filepath.Join(filename, "bc", "0")) {
		system = "openvz"
		role = "host"
	} else if PathExists(filepath.Join(filename, "vz")) {
		system = "openvz"
		role = "guest"
	}

	// not use dmidecode because it requires root
	if PathExists(filepath.Join(filename, "self", "status")) {
		ok, _, _ := FileContainsAny(filepath.Join(filename, "self", "status"), "s_context:,VxID:")
		if ok {
			system = "linux-vserver"
			// TODO: guest or host
		}
	}

	if PathExists(filepath.Join(filename, "1", "environ")) {
		contents, err := ReadFile(filepath.Join(filename, "1", "environ"))

		if err == nil {
			if strings.Contains(contents, "container=lxc") {
				system = "lxc"
				role = "guest"
			}
		}
	}

	if PathExists(filepath.Join(filename, "self", "cgroup")) {
		ok, match, _ := FileContainsAny(filepath.Join(filename, "self", "cgroup"), "lxc,docker,machine-rkt")
		if ok {
			switch match {
			case "lxc":
				system = "lxc"
				role = "guest"
			case "docker":
				system = "docker"
				role = "guest"
			case "machine-rkt":
				system = "rkt"
				role = "guest"
			}
		}
	}

	if PathExists(HostEtcWithContext(ctx, "os-release")) {
		p, _, err := GetOSReleaseWithContext(ctx)
		if err == nil && p == "coreos" {
			system = "rkt" // Is it true?
			role = "host"
		}
	}

	if PathExists(HostRootWithContext(ctx, ".dockerenv")) {
		system = "docker"
		role = "guest"
	}

	// before returning for the first time, cache the system and role
	cachedVirtOnce.Do(func() {
		cachedVirtMutex.Lock()
		defer cachedVirtMutex.Unlock()
		cachedVirtMap = map[string]string{
			"system": system,
			"role":   role,
		}
	})

	return system, role, nil
}

func GetOSRelease() (platform, version string, err error) {
	return GetOSReleaseWithContext(context.Background())
}

func GetOSReleaseWithContext(ctx context.Context) (platform, version string, err error) {
	lines, _, err := ReadLines(HostEtcWithContext(ctx, "os-release"))
	if err != nil {
		return "", "", nil // return empty
	}
	for line := range lines {
		if strings.Count(line, "=") < 1 {
			continue
		}
		key, value, _ := strings.Cut(line, "=")
		switch key {
		case "ID": // use ID for lowercase
			platform = trimQuotes(value)
		case "VERSION_ID":
			version = trimQuotes(value)
		}
	}

	// cleanup amazon ID
	if platform == "amzn" {
		platform = "amazon"
	}

	return platform, version, nil
}

// Remove quotes of the source string
func trimQuotes(s string) string {
	if len(s) >= 2 {
		if s[0] == '"' && s[len(s)-1] == '"' {
			return s[1 : len(s)-1]
		}
	}
	return s
}
