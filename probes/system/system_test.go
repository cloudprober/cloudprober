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
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/system/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
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

	// 7. meminfo
	memInfoContent := `MemTotal:       16000000 kB
MemFree:         8000000 kB
MemAvailable:   12000000 kB
Buffers:          500000 kB
Cached:          2000000 kB
`
	if err := os.WriteFile(filepath.Join(tmpDir, "meminfo"), []byte(memInfoContent), 0644); err != nil {
		t.Fatal(err)
	}

	// 8. diskstats
	// 8 0 sda 100 200 300 400 0 0 0 0 0 0 0 ...
	// Need 14+ fields
	diskStatsContent := "   8       0 sda 100 200 500 400 10 20 600 40 0 0 0\n   8 16 sdb 0 0 0 0 0 0 0 0 0 0 0\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "diskstats"), []byte(diskStatsContent), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Probe{
		name:   "test_probe",
		c:      &configpb.ProbeConf{},
		l:      &logger.Logger{},
		sysDir: tmpDir,
		diskUsageFunc: func(path string) (uint64, uint64, error) {
			if path == "/" {
				return 1000000, 400000, nil // Total, Free
			}
			return 0, 0, os.ErrNotExist
		},
		opts: &options.Options{
			ProbeConf: &configpb.ProbeConf{},
		},
	}

	em := metrics.NewEventMetrics(time.Now())
	em.Kind = metrics.GAUGE
	emCum := metrics.NewEventMetrics(time.Now())
	emCum.Kind = metrics.CUMULATIVE

	// Test Global Metrics (includes Memory)
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

	// Memory
	assert.Equal(t, 16000000.0*1024, metricsMap["system_mem_total"])
	assert.Equal(t, 8000000.0*1024, metricsMap["system_mem_free"])

	// Verify Cumulative Metrics
	metricsMapCum := make(map[string]float64)
	for _, m := range emCum.MetricsKeys() {
		val := emCum.Metric(m).(*metrics.Float).Float64()
		metricsMapCum[m] = val
	}
	assert.Equal(t, 1000.0, metricsMapCum["system_procs_total"])

	// Test Disk Usage
	dataChan := make(chan *metrics.EventMetrics, 10)
	p.exportDiskUsageStats(time.Now(), dataChan)

	select {
	case emDU := <-dataChan:
		assert.Equal(t, "system", emDU.Label("ptype"))
		assert.Equal(t, "/", emDU.Label("mount_point"))

		assert.Equal(t, int64(1000000), emDU.Metric("system_disk_total").(*metrics.Int).Int64())
		assert.Equal(t, int64(400000), emDU.Metric("system_disk_free").(*metrics.Int).Int64())
		assert.Nil(t, emDU.Metric("system_disk_used"))         // Used not exported
		assert.Nil(t, emDU.Metric("system_disk_used_percent")) // Percent not exported

	default:
		t.Error("expected disk usage metrics")
	}

	// Test Disk IO
	p.exportDiskIOStats(time.Now(), dataChan)

	// Expect aggregated stats by default (DeviceStats default: aggregated=true, individual=false)
	timeout := time.After(1 * time.Second)
	// We expect 3 events now: sda, sdb, and aggregated
	// But in default config (nil or empty DeviceStats), we might only get aggregated if I implemented default=true correctly.
	// Let's check logic:
	// if p.c.GetDiskIoStats().GetExportIndividualStats() { ... } -> Default false
	// if p.c.GetDiskIoStats().GetExportAggregatedStats() { ... } -> Default true

	// So by default, we only get aggregated.
	select {
	case emIO := <-dataChan:
		// Aggregated
		assert.Equal(t, metrics.Kind(metrics.CUMULATIVE), emIO.Kind)
		assert.Equal(t, "system", emIO.Label("ptype"))
		assert.Equal(t, "", emIO.Label("device")) // No device label for aggregated

		val := emIO.Metric("system_disk_read_bytes").(*metrics.Float).Float64()
		assert.Equal(t, 500.0*512, val)
	case <-timeout:
		t.Fatal("timeout waiting for disk IO stats")
	}
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

func TestExportNetDevStats(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock files
	if err := os.MkdirAll(filepath.Join(tmpDir, "net"), 0755); err != nil {
		t.Fatal(err)
	}

	netDevContent := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0:    1000      10    1    2    0     0          0         0     2000      20    0    0    0     0       0          0
  eth1:     500       5    0    1    0     0          0         0     1000      10    0    0    0     0       0          0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "net/dev"), []byte(netDevContent), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Probe{
		name:   "test_probe",
		c:      &configpb.ProbeConf{},
		l:      &logger.Logger{},
		sysDir: tmpDir,
		opts: &options.Options{
			ProbeConf: &configpb.ProbeConf{},
		},
	}

	// Enable individual stats for test
	p.c.NetDevStats = &configpb.ResourceUsage{
		ExportIndividualStats: proto.Bool(true),
		ExportAggregatedStats: proto.Bool(true),
	}

	dataChan := make(chan *metrics.EventMetrics, 10)
	p.exportNetDevStats(time.Now(), dataChan)

	// Expect 3 metrics: eth0, eth1, and aggregated
	foundEth0 := false
	foundEth1 := false
	foundAgg := false

	for i := 0; i < 3; i++ {
		select {
		case em := <-dataChan:
			if em.Label("iface") == "eth0" {
				foundEth0 = true
				valMap := make(map[string]float64)
				for _, m := range em.MetricsKeys() {
					valMap[m] = em.Metric(m).(*metrics.Float).Float64()
				}
				assert.Equal(t, 1000.0, valMap["system_net_rx_bytes"])
			} else if em.Label("iface") == "eth1" {
				foundEth1 = true
				valMap := make(map[string]float64)
				for _, m := range em.MetricsKeys() {
					valMap[m] = em.Metric(m).(*metrics.Float).Float64()
				}
				assert.Equal(t, 500.0, valMap["system_net_rx_bytes"])
			} else {
				// Aggregated
				foundAgg = true
				assert.Equal(t, "", em.Label("iface"))
				valMap := make(map[string]float64)
				for _, m := range em.MetricsKeys() {
					valMap[m] = em.Metric(m).(*metrics.Float).Float64()
				}
				assert.Equal(t, 1500.0, valMap["system_net_rx_bytes"])
				assert.Equal(t, 3000.0, valMap["system_net_tx_bytes"])
			}
		default:
			t.Fatal("expected more metrics")
		}
	}
	assert.True(t, foundEth0, "found eth0")
	assert.True(t, foundEth1, "found eth1")
	assert.True(t, foundAgg, "found aggregated")
}

func TestExportDiskUsageStats_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()

	// Create mock mounts file
	mountsContent := `/dev/root / ext4 rw 0 0
/dev/sdb1 /data ext4 rw 0 0
proc /proc proc rw 0 0
sysfs /sys/kernel/debug sysfs rw 0 0
nsfs /run/netns/cni-123 nsfs rw 0 0
devtmpfs /dev devtmpfs rw 0 0
snap /snap/core/123 squashfs ro 0 0
`
	if err := os.WriteFile(filepath.Join(tmpDir, "mounts"), []byte(mountsContent), 0644); err != nil {
		t.Fatal(err)
	}

	p := &Probe{
		name:   "test_probe",
		c:      &configpb.ProbeConf{},
		l:      &logger.Logger{},
		sysDir: tmpDir,
		diskUsageFunc: func(path string) (uint64, uint64, error) {
			if path == "/" {
				return 1000, 400, nil
			}
			if path == "/data" {
				return 2000, 1000, nil
			}
			if strings.HasPrefix(path, "/sys/") || strings.HasPrefix(path, "/proc") || strings.HasPrefix(path, "/dev") || strings.HasPrefix(path, "/run/netns") || strings.HasPrefix(path, "/snap/") {
				return 100, 10, nil
			}
			return 0, 0, os.ErrNotExist
		},
		opts: &options.Options{
			ProbeConf: &configpb.ProbeConf{},
		},
	}

	// Enable config with regex
	p.c.DiskUsageStats = &configpb.ResourceUsage{
		IncludeNameRegex:      proto.String("^/"),
		ExcludeNameRegex:      proto.String("^/proc"),
		ExportIndividualStats: proto.Bool(true),
		ExportAggregatedStats: proto.Bool(true),
	}

	dataChan := make(chan *metrics.EventMetrics, 10)
	p.exportDiskUsageStats(time.Now(), dataChan)

	// Expect /, /data, and aggregated
	// /: total 1000
	// /data: total 2000
	// agg: total 3000

	foundRoot := false
	foundData := false
	foundAgg := false

	for i := 0; i < 3; i++ {
		select {
		case em := <-dataChan:
			if em.Label("mount_point") == "/" {
				foundRoot = true
				if val := em.Metric("system_disk_total").(*metrics.Int).Int64(); val != 1000 {
					t.Errorf("Root total = %d, want 1000", val)
				}
				if val := em.Metric("system_disk_free").(*metrics.Int).Int64(); val != 400 {
					t.Errorf("Root free = %d, want 400", val)
				}
			} else if em.Label("mount_point") == "/data" {
				foundData = true
				if val := em.Metric("system_disk_total").(*metrics.Int).Int64(); val != 2000 {
					t.Errorf("Data total = %d, want 2000", val)
				}
				if val := em.Metric("system_disk_free").(*metrics.Int).Int64(); val != 1000 {
					t.Errorf("Data free = %d, want 1000", val)
				}
			} else {
				// Verify we don't get excluded mount points
				mp := em.Label("mount_point")
				for _, exclude := range []string{"/dev", "/sys", "/proc", "/run/netns", "/snap"} {
					if mp == exclude || strings.HasPrefix(mp, exclude+"/") {
						t.Errorf("Got excluded mount point: %s", mp)
					}
				}

				// Aggregated has no mount_point label (empty string if requested? no label usually)
				// My impl doesn't add label for aggregated.
				if em.Label("mount_point") == "" {
					foundAgg = true
					if val := em.Metric("system_disk_total").(*metrics.Int).Int64(); val != 3000 {
						t.Errorf("Agg total = %d, want 3000", val)
					}
					if val := em.Metric("system_disk_free").(*metrics.Int).Int64(); val != 1400 {
						t.Errorf("Agg free = %d, want 1400", val)
					}
				}
			}
		case <-time.After(time.Second):
			t.Fatal("timeout")
		}
	}
	assert.True(t, foundRoot, "found root")
	assert.True(t, foundData, "found data")
	assert.True(t, foundAgg, "found aggregated")
}

func TestMatchDevice(t *testing.T) {
	tests := []struct {
		name    string
		dev     string
		include string
		exclude string
		want    bool
	}{
		{"empty", "eth0", "", "", true},
		{"include_match", "eth0", "^eth", "", true},
		{"include_miss", "lo", "^eth", "", false},
		{"exclude_match", "eth0", "", "^eth", false},
		{"exclude_miss", "lo", "", "^eth", true},
		{"both_include", "eth0", "^eth", "^lo", true},
		{"both_exclude", "eth0", "^eth", "0$", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := matchDevice(tt.dev, tt.include, tt.exclude); got != tt.want {
				t.Errorf("matchDevice() = %v, want %v", got, tt.want)
			}
		})
	}
}
