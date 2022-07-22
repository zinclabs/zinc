/* Copyright 2022 Zinc Labs Inc. and Contributors
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
 */

package core

import (
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/robfig/cron/v3"
	"github.com/rs/zerolog/log"
	"github.com/segmentio/analytics-go/v3"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"

	"github.com/zinclabs/zinc/pkg/config"
	"github.com/zinclabs/zinc/pkg/errors"
	"github.com/zinclabs/zinc/pkg/ider"
	"github.com/zinclabs/zinc/pkg/meta"
	"github.com/zinclabs/zinc/pkg/metadata"
)

// Telemetry instance
var Telemetry = newTelemetry()

type telemetry struct {
	instanceID   string
	events       chan analytics.Track
	baseInfo     map[string]interface{}
	baseInfoOnce sync.Once
}

func newTelemetry() *telemetry {
	t := new(telemetry)
	t.events = make(chan analytics.Track, 100)
	t.initBaseInfo()

	if config.Global.TelemetryEnable {
		go t.runEvents()
	}

	return t
}

func (t *telemetry) createInstanceID() string {
	instanceID := ider.Generate()
	_ = metadata.KV.Set("instance_id", []byte(instanceID))
	return instanceID
}

func (t *telemetry) GetInstanceID() string {
	if t.instanceID != "" {
		return t.instanceID
	}

	val, err := metadata.KV.Get("instance_id")
	if err != nil {
		if err != errors.ErrKeyNotFound {
			log.Error().Err(err).Msg("core.Telemetry.GetInstanceID: error accessing stored fields")
		}
	}
	if val != nil {
		t.instanceID = string(val)
	}
	if t.instanceID == "" {
		t.instanceID = t.createInstanceID()
	}
	return t.instanceID
}

func (t *telemetry) initBaseInfo() {
	t.baseInfoOnce.Do(func() {
		m, _ := mem.VirtualMemory()
		cpuCount, _ := cpu.Counts(true)
		zone, _ := time.Now().Local().Zone()

		t.baseInfo = map[string]interface{}{
			"os":           runtime.GOOS,
			"arch":         runtime.GOARCH,
			"zinc_version": meta.Version,
			"time_zone":    zone,
			"cpu_count":    cpuCount,
			"total_memory": m.Total / 1024 / 1024,
		}
	})
}

func (t *telemetry) Instance() {
	if !config.Global.TelemetryEnable {
		return
	}

	traits := analytics.NewTraits().
		Set("index_count", ZINC_INDEX_LIST.Len()).
		Set("total_index_size_mb", t.TotalIndexSize())

	for k, v := range t.baseInfo {
		traits.Set(k, v)
	}

	_ = meta.SEGMENT_CLIENT.Enqueue(analytics.Identify{
		UserId: t.GetInstanceID(),
		Traits: traits,
	})
}

func (t *telemetry) Event(event string, data map[string]interface{}) {
	if !config.Global.TelemetryEnable {
		return
	}

	props := analytics.NewProperties()
	for k, v := range t.baseInfo {
		props.Set(k, v)
	}
	for k, v := range data {
		props.Set(k, v)
	}

	t.events <- analytics.Track{
		UserId:     t.GetInstanceID(),
		Event:      event,
		Properties: props,
	}
}

func (t *telemetry) runEvents() {
	for event := range t.events {
		_ = meta.SEGMENT_CLIENT.Enqueue(event)
	}
}

func (t *telemetry) TotalIndexSize() uint64 {
	TotalIndexSize := uint64(0)
	for _, idx := range ZINC_INDEX_LIST.List() {
		TotalIndexSize += t.GetIndexSize(idx.GetName())
	}
	return TotalIndexSize
}

func (t *telemetry) GetIndexSize(indexName string) uint64 {
	if index, ok := ZINC_INDEX_LIST.Get(indexName); ok {
		return atomic.LoadUint64(&index.ref.Stats.StorageSize) / 1024 / 1024 // convert to MB
	}
	return 0
}

func (t *telemetry) HeartBeat() {
	m, err := mem.VirtualMemory()
	if err != nil {
		log.Error().Err(err).Msg("core.Telemetry.HeartBeat: error getting memory info")
		return
	}
	data := make(map[string]interface{})
	data["index_count"] = ZINC_INDEX_LIST.Len()
	data["total_index_size_mb"] = t.TotalIndexSize()
	data["memory_used_percent"] = m.UsedPercent
	t.Event("heartbeat", data)
}

func (t *telemetry) Cron() {
	if !config.Global.TelemetryEnable {
		return
	}

	c := cron.New()
	_, _ = c.AddFunc("@every 30m", t.HeartBeat)
	c.Start()
}
