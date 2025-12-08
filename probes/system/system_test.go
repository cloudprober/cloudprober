// Copyright 2025 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package system

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/system/proto"
	"github.com/stretchr/testify/assert"
)

func TestProbeExportMetrics(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock files
	// 1. sys/fs/file-nr
	if err := os.MkdirAll(filepath.Join(tmpDir, "sys/fs"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "sys/fs/file-nr"), []byte("100 0 1000"), 0644); err != nil {
		t.Fatal(err)
	}

	// 2. stat
	if err := os.WriteFile(filepath.Join(tmpDir, "stat"), []byte("procs_running 5\nprocs_blocked 1\nprocesses 1000\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 3. net/sockstat
	if err := os.MkdirAll(filepath.Join(tmpDir, "net"), 0755); err != nil {
		t.Fatal(err)
	}
	// "TCP: inuse 1 orphan 0 tw 0 alloc 1 mem 1"
	if err := os.WriteFile(filepath.Join(tmpDir, "net/sockstat"), []byte("sockets: used 123\nTCP: inuse 10 orphan 0 tw 0 alloc 1 mem 1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// 4. net/dev
	netDevContent := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0:    1000      10    1    2    0     0          0         0     2000      20    0    0    0     0       0          0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "net/dev"), []byte(netDevContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 5. uptime
	if err := os.WriteFile(filepath.Join(tmpDir, "uptime"), []byte("9876.54 1234.56"), 0644); err != nil {
		t.Fatal(err)
	}

	// 6. loadavg
	if err := os.WriteFile(filepath.Join(tmpDir, "loadavg"), []byte("0.50 0.40 0.30 1/100 12345"), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Probe{
		name:   "test_probe",
		c:      &configpb.ProbeConf{},
		l:      &logger.Logger{},
		sysDir: tmpDir,
	}

	em := metrics.NewEventMetrics(time.Now())
	em.Kind = metrics.GAUGE
	emCum := metrics.NewEventMetrics(time.Now())
	emCum.Kind = metrics.CUMULATIVE

	p.exportGlobalMetrics(em, emCum)

	// Verify Gauge Metrics
	metricsMap := make(map[string]float64)
	for _, m := range em.MetricsKeys() {
		val := em.Metric(m).(*metrics.Float).Float64()
		metricsMap[m] = val
	}

	assert.Equal(t, 100.0, metricsMap["system_file_descriptors_allocated"])
	assert.Equal(t, 1000.0, metricsMap["system_file_descriptors_max"])

	assert.Equal(t, 5.0, metricsMap["system_procs_running"])
	assert.Equal(t, 1.0, metricsMap["system_procs_blocked"])
	// system_procs_total is CUMULATIVE now

	assert.Equal(t, 123.0, metricsMap["system_sockets_in_use"])
	assert.Equal(t, 10.0, metricsMap["system_sockets_tcp_inuse"])

	assert.InDelta(t, 9876.54, metricsMap["system_uptime_sec"], 0.001)

	assert.Equal(t, 0.50, metricsMap["system_load_1m"])
	assert.Equal(t, 0.40, metricsMap["system_load_5m"])
	assert.Equal(t, 0.30, metricsMap["system_load_15m"])

	// Verify Cumulative Metrics
	metricsMapCum := make(map[string]float64)
	for _, m := range emCum.MetricsKeys() {
		val := emCum.Metric(m).(*metrics.Float).Float64()
		metricsMapCum[m] = val
	}
	assert.Equal(t, 1000.0, metricsMapCum["system_procs_total"])
}

func TestInit(t *testing.T) {
	p := &Probe{}
	opts := &options.Options{
		ProbeConf: &configpb.ProbeConf{},
	}
	err := p.Init("test", opts)
	assert.NoError(t, err)
	assert.Equal(t, "/proc", p.sysDir)
}
