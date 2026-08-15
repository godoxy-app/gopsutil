// SPDX-License-Identifier: BSD-3-Clause
package net

import (
	"errors"
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/shirou/gopsutil/v4/internal/common"
)

func TestAddrString(t *testing.T) {
	v := Addr{IP: "192.168.0.1", Port: 8000}

	s := v.String()
	assert.JSONEqf(t, `{"ip":"192.168.0.1","port":8000}`, s, "Addr string is invalid: %v", v)
}

func TestIOCountersStatString(t *testing.T) {
	v := IOCountersStat{
		BytesSent: 100,
	}
	e := `{"bytes_sent":100,"bytes_recv":0,"upload_speed":0.0,"download_speed":0.0}`
	assert.JSONEqf(t, e, v.String(), "NetIOCountersStat string is invalid: %v", v)
}

func TestProtoCountersStatString(t *testing.T) {
	v := ProtoCountersStat{
		Protocol: "tcp",
		Stats: map[string]int64{
			"MaxConn":      -1,
			"ActiveOpens":  4000,
			"PassiveOpens": 3000,
		},
	}
	e := `{"protocol":"tcp","stats":{"ActiveOpens":4000,"MaxConn":-1,"PassiveOpens":3000}}`
	assert.JSONEqf(t, e, v.String(), "NetProtoCountersStat string is invalid: %v", v)
}

func TestConnectionStatString(t *testing.T) {
	v := ConnectionStat{
		Fd:     10,
		Family: 10,
		Type:   10,
		Uids:   []int32{10, 10},
	}
	e := `{"fd":10,"family":10,"type":10,"localaddr":{"ip":"","port":0},"remoteaddr":{"ip":"","port":0},"status":"","uids":[10,10],"pid":0}`
	assert.JSONEqf(t, e, v.String(), "NetConnectionStat string is invalid: %v", v)
}

func TestIOCountersAll(t *testing.T) {
	v, err := IOCounters(false)
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoErrorf(t, err, "Could not get NetIOCounters: %v", err)
	_, err = IOCounters(true)
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoErrorf(t, err, "Could not get NetIOCounters: %v", err)
	assert.Lenf(t, v, 1, "Could not get NetIOCounters: %v", v)
}

func TestIOCountersPerNic(t *testing.T) {
	v, err := IOCounters(true)
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoErrorf(t, err, "Could not get NetIOCounters: %v", err)
	assert.NotEmptyf(t, v, "Could not get NetIOCounters: %v", v)
}

func TestInterfaces(t *testing.T) {
	v, err := Interfaces()
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoErrorf(t, err, "Could not get NetInterfaceStat: %v", err)
	assert.NotEmptyf(t, v, "Could not get NetInterfaceStat: %v", err)
	for _, vv := range v {
		assert.NotEmptyf(t, vv.Name, "Invalid NetInterface: %v", vv)
	}
}

func TestProtoCountersStatsAll(t *testing.T) {
	v, err := ProtoCounters(nil)
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoErrorf(t, err, "Could not get NetProtoCounters: %v", err)
	require.NotEmptyf(t, v, "Could not get NetProtoCounters: %v", err)
	for _, vv := range v {
		assert.NotEmptyf(t, vv.Protocol, "Invalid NetProtoCountersStat: %v", vv)
		assert.NotEmptyf(t, vv.Stats, "Invalid NetProtoCountersStat: %v", vv)
	}
}

func TestProtoCountersStats(t *testing.T) {
	v, err := ProtoCounters([]string{"tcp", "ip"})
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoErrorf(t, err, "Could not get NetProtoCounters: %v", err)
	require.NotEmptyf(t, v, "Could not get NetProtoCounters: %v", err)
	require.Lenf(t, v, 2, "Go incorrect number of NetProtoCounters: %v", err)
	for _, vv := range v {
		if vv.Protocol != "tcp" && vv.Protocol != "ip" {
			t.Errorf("Invalid NetProtoCountersStat: %v", vv)
		}
		assert.NotEmptyf(t, vv.Stats, "Invalid NetProtoCountersStat: %v", vv)
	}
}

func TestConnections(t *testing.T) {
	if ci := os.Getenv("CI"); ci != "" { // skip if test on CI
		return
	}

	v, err := Connections("inet")
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoErrorf(t, err, "could not get NetConnections: %v", err)
	assert.NotEmptyf(t, v, "could not get NetConnections: %v", v)
	for _, vv := range v {
		assert.NotZerof(t, vv.Family, "invalid NetConnections: %v", vv)
	}
}

func TestFilterCounters(t *testing.T) {
	if ci := os.Getenv("CI"); ci != "" { // skip if test on CI
		return
	}

	if runtime.GOOS == "linux" {
		// some test environment has not the path.
		if !common.PathExists("/proc/sys/net/netfilter/nf_connTrackCount") {
			t.SkipNow()
		}
	}

	v, err := FilterCounters()
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoErrorf(t, err, "could not get NetConnections: %v", err)
	assert.NotEmptyf(t, v, "could not get NetConnections: %v", v)
	for _, vv := range v {
		assert.NotZerof(t, vv.ConnTrackMax, "nf_connTrackMax needs to be greater than zero: %v", vv)
	}
}

func TestInterfaceStatString(t *testing.T) {
	v := InterfaceStat{
		Index:        0,
		MTU:          1500,
		Name:         "eth0",
		HardwareAddr: "01:23:45:67:89:ab",
		Flags:        []string{"up", "down"},
		Addrs:        InterfaceAddrList{{Addr: "1.2.3.4"}, {Addr: "5.6.7.8"}},
	}

	s := v.String()
	assert.JSONEqf(t, `{"index":0,"mtu":1500,"name":"eth0","hardwareAddr":"01:23:45:67:89:ab","flags":["up","down"],"addrs":[{"addr":"1.2.3.4"},{"addr":"5.6.7.8"}]}`, s, "InterfaceStat string is invalid: %v", s)

	list := InterfaceStatList{v, v}
	s = list.String()
	assert.JSONEqf(t, `[{"index":0,"mtu":1500,"name":"eth0","hardwareAddr":"01:23:45:67:89:ab","flags":["up","down"],"addrs":[{"addr":"1.2.3.4"},{"addr":"5.6.7.8"}]},{"index":0,"mtu":1500,"name":"eth0","hardwareAddr":"01:23:45:67:89:ab","flags":["up","down"],"addrs":[{"addr":"1.2.3.4"},{"addr":"5.6.7.8"}]}]`, s, "InterfaceStatList string is invalid: %v", s)
}
