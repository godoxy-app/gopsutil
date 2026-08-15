// SPDX-License-Identifier: BSD-3-Clause

package sensors

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yusing/goutils/intern"
	"github.com/yusing/goutils/num"

	"github.com/shirou/gopsutil/v4/internal/common"
)

func TestTemperatureStat_String(t *testing.T) {
	v := TemperatureStat{
		SensorKey:   intern.MakeValue("CPU"),
		Temperature: 1.1,
	}
	loadOrStoreSensorTemps(v.SensorKey, num.NewPercentage(30.0), num.NewPercentage(0.4))
	s := `{"name":"CPU","temperature":1.1,"high":30.0,"critical":0.4}`
	assert.Equalf(t, s, v.String(), "TemperatureStat string is invalid, %v", v.String())
}

func TestTemperatures(t *testing.T) {
	if os.Getenv("CI") != "" {
		t.Skip("Skip CI")
	}
	v, err := SensorsTemperatures()
	if errors.Is(err, common.ErrNotImplementedError) {
		t.Skip("not implemented")
	}
	require.NoError(t, err)
	assert.NotEmptyf(t, v, "Could not get temperature %v", v)
	t.Log(v)
}
