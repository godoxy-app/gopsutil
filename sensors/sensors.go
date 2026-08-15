// SPDX-License-Identifier: BSD-3-Clause

package sensors

import (
	"context"
	"encoding/json"
	"sync/atomic"

	"github.com/puzpuzpuz/xsync/v4"
	"github.com/shirou/gopsutil/v4/internal/common"
	"github.com/yusing/goutils/num"
)

type Warnings = common.Warnings

var invoke common.Invoker = common.Invoke{}

const maxSensorKeys = 256

var (
	sensorKeysIndexes  = xsync.NewMap[string, uint8](xsync.WithGrowOnly(), xsync.WithPresize(maxSensorKeys)) // allocates only once per unique sensor key
	sensorTempHigh     [maxSensorKeys]num.Percentage
	sensorTempCritical [maxSensorKeys]num.Percentage
	sensorKeysIdx      atomic.Uint32
)

//go:nosplit
func loadOrStoreSensorTemps(sensorKey string, high, critical num.Percentage) {
	idx, loaded := sensorKeysIndexes.LoadOrCompute(sensorKey, func() (uint8, bool) {
		return uint8(sensorKeysIdx.Add(1)), false
	})
	if !loaded {
		sensorTempHigh[idx] = high
		sensorTempCritical[idx] = critical
	}
}

//go:nosplit
func loadOrStoreSensorTempsCompute(sensorKey string, high, critical func() num.Percentage) {
	idx, loaded := sensorKeysIndexes.LoadOrCompute(sensorKey, func() (uint8, bool) {
		return uint8(sensorKeysIdx.Add(1)), false
	})
	if !loaded {
		sensorTempHigh[idx] = high()
		sensorTempCritical[idx] = critical()
	}
}

type TemperatureStat struct {
	SensorKey   string                             `json:"name"`
	Temperature float32                            `json:"temperature"`
	High_       common.PlaceHolder[num.Percentage] `json:"high" swaggertype:"number"`
	Critical_   common.PlaceHolder[num.Percentage] `json:"critical" swaggertype:"number"`
}

type temperatureStatFull struct {
	SensorKey   string         `json:"name"`
	Temperature float32        `json:"temperature"`
	High        num.Percentage `json:"high"`
	Critical    num.Percentage `json:"critical"`
}

func (t TemperatureStat) String() string {
	s, _ := t.MarshalJSON()
	return string(s)
}

func (t TemperatureStat) MarshalJSON() ([]byte, error) {
	k := t.SensorKey
	// if !ok, idx will be 0 and we don't care
	idx, _ := sensorKeysIndexes.Load(k)
	high := sensorTempHigh[idx]
	critical := sensorTempCritical[idx]
	return json.Marshal(temperatureStatFull{
		SensorKey:   k,
		Temperature: t.Temperature,
		High:        high,
		Critical:    critical,
	})
}

func (t *TemperatureStat) UnmarshalJSON(data []byte) error {
	var v temperatureStatFull
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	loadOrStoreSensorTemps(v.SensorKey, v.High, v.Critical)
	t.SensorKey = common.InternString(v.SensorKey)
	t.Temperature = v.Temperature
	return nil
}

func SensorsTemperatures() ([]TemperatureStat, error) {
	return TemperaturesWithContext(context.Background())
}
