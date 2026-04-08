// Copyright 2025-2026 The Cloudprober Authors.
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

//go:build linux
// +build linux

package system

import (
	"context"
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

// setupMockProcDir creates a temporary directory with mock /proc files.
// Optional overrides can replace default file contents (keyed by relative path).
func setupMockProcDir(t *testing.T, overrides map[string]string) string {
	t.Helper()
	tmpDir := t.TempDir()

	defaults := map[string]string{
		"sys/fs/file-nr": "100 0 1000",
		"stat":           "procs_running 5\nprocs_blocked 1\nprocesses 1000\n",
		"net/sockstat":   "sockets: used 123\nTCP: inuse 10 orphan 0 tw 0 alloc 1 mem 1\n",
		"net/dev": `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0:    1000      10    1    2    0     0          0         0     2000      20    0    0    0     0       0          0
`,
		"uptime":  "9876.54 1234.56",
		"loadavg": "0.50 0.40 0.30 1/100 12345",
		"meminfo": `MemTotal:       16000000 kB
MemFree:         8000000 kB
MemAvailable:   12000000 kB
Buffers:          500000 kB
Cached:          2000000 kB
`,
		"diskstats": "   8       0 sda 100 200 500 400 10 20 600 40 0 0 0\n",
		"mounts":    "/dev/root / ext4 rw 0 0\n",
	}

	for k, v := range overrides {
		defaults[k] = v
	}

	for relPath, content := range defaults {
		fullPath := filepath.Join(tmpDir, relPath)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return tmpDir
}

// newTestProbe creates a Probe with mock proc dir and disk usage function.
func newTestProbe(t *testing.T, tmpDir string, conf *configpb.ProbeConf) *Probe {
	t.Helper()
	if conf == nil {
		conf = &configpb.ProbeConf{}
	}
	return &Probe{
		name:   "test_probe",
		c:      conf,
		l:      &logger.Logger{},
		sysDir: tmpDir,
		diskUsageFunc: func(path string) (uint64, uint64, error) {
			switch path {
			case "/":
				return 1000000, 400000, nil
			case "/data":
				return 2000, 1000, nil
			default:
				return 0, 0, os.ErrNotExist
			}
		},
		opts: &options.Options{
			ProbeConf: conf,
		},
		diskErrMounts: make(map[string]bool),
	}
}

// drainMetrics reads all available EventMetrics from a channel.
func drainMetrics(ch chan *metrics.EventMetrics) []*metrics.EventMetrics {
	var result []*metrics.EventMetrics
	for {
		select {
		case em := <-ch:
			result = append(result, em)
		default:
			return result
		}
	}
}

// floatMetricsMap extracts all float metric values from an EventMetrics.
func floatMetricsMap(em *metrics.EventMetrics) map[string]float64 {
	m := make(map[string]float64)
	for _, k := range em.MetricsKeys() {
		if f, ok := em.Metric(k).(*metrics.Float); ok {
			m[k] = f.Float64()
		}
	}
	return m
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

func TestExportGlobalMetrics(t *testing.T) {
	tmpDir := setupMockProcDir(t, nil)
	p := newTestProbe(t, tmpDir, nil)

	em := metrics.NewEventMetrics(time.Now())
	em.Kind = metrics.GAUGE
	emCum := metrics.NewEventMetrics(time.Now())
	emCum.Kind = metrics.CUMULATIVE

	p.exportGlobalMetrics(em, emCum)

	gauge := floatMetricsMap(em)
	cum := floatMetricsMap(emCum)

	// File descriptors
	assert.Equal(t, 100.0, gauge["system_file_descriptors_allocated"])
	assert.Equal(t, 1000.0, gauge["system_file_descriptors_max"])

	// Process stats
	assert.Equal(t, 5.0, gauge["system_procs_running"])
	assert.Equal(t, 1.0, gauge["system_procs_blocked"])
	assert.Equal(t, 1000.0, cum["system_procs_total"])

	// Sockets
	assert.Equal(t, 123.0, gauge["system_sockets_inuse"])
	assert.Equal(t, 10.0, gauge["system_sockets_tcp_inuse"])

	// Uptime
	assert.InDelta(t, 9876.54, gauge["system_uptime_sec"], 0.001)

	// Load average
	assert.Equal(t, 0.50, gauge["system_load_1m"])
	assert.Equal(t, 0.40, gauge["system_load_5m"])
	assert.Equal(t, 0.30, gauge["system_load_15m"])

	// Memory
	assert.Equal(t, 16000000.0*1024, gauge["system_mem_total"])
	assert.Equal(t, 8000000.0*1024, gauge["system_mem_free"])
	assert.Equal(t, 12000000.0*1024, gauge["system_mem_available"])
	assert.Equal(t, 500000.0*1024, gauge["system_mem_buffers"])
	assert.Equal(t, 2000000.0*1024, gauge["system_mem_cached"])
}

func TestExportNetDevStats(t *testing.T) {
	twoIfaceNetDev := `Inter-|   Receive                                                |  Transmit
 face |bytes    packets errs drop fifo frame compressed multicast|bytes    packets errs drop fifo colls carrier compressed
  eth0:    1000      10    1    2    0     0          0         0     2000      20    0    0    0     0       0          0
  eth1:     500       5    0    1    0     0          0         0     1000      10    0    0    0     0       0          0
`
	threeIfaceNetDev := twoIfaceNetDev + "    lo:     100       1    0    0    0     0          0         0      100       1    0    0    0     0       0          0\n"

	tests := []struct {
		name       string
		netDev     string
		config     *configpb.ResourceUsage
		wantIfaces []string // expected individual iface labels
		wantAgg    map[string]float64
		wantCount  int // total expected EventMetrics
	}{
		{
			name:   "individual_and_aggregated",
			netDev: twoIfaceNetDev,
			config: &configpb.ResourceUsage{
				ExportIndividualStats: proto.Bool(true),
				ExportAggregatedStats: proto.Bool(true),
			},
			wantIfaces: []string{"eth0", "eth1"},
			wantAgg: map[string]float64{
				"system_net_aggregated_rx_bytes": 1500,
				"system_net_aggregated_tx_bytes": 3000,
			},
			wantCount: 3,
		},
		{
			name:   "filter_before_aggregation",
			netDev: threeIfaceNetDev,
			config: &configpb.ResourceUsage{
				IncludeNameRegex:      proto.String("^eth"),
				ExportIndividualStats: proto.Bool(true),
				ExportAggregatedStats: proto.Bool(true),
			},
			wantIfaces: []string{"eth0", "eth1"},
			wantAgg: map[string]float64{
				"system_net_aggregated_rx_bytes": 1500, // lo excluded
				"system_net_aggregated_tx_bytes": 3000,
			},
			wantCount: 3, // eth0, eth1, aggregated (no lo)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupMockProcDir(t, map[string]string{"net/dev": tt.netDev})
			p := newTestProbe(t, tmpDir, nil)
			p.c.NetDevStats = tt.config

			dataChan := make(chan *metrics.EventMetrics, 10)
			p.exportNetDevStats(time.Now(), dataChan)
			results := drainMetrics(dataChan)

			assert.Len(t, results, tt.wantCount)

			foundIfaces := make(map[string]bool)
			var aggEM *metrics.EventMetrics
			for _, em := range results {
				iface := em.Label("iface")
				if iface != "" {
					foundIfaces[iface] = true
				} else {
					aggEM = em
				}
			}

			for _, iface := range tt.wantIfaces {
				assert.True(t, foundIfaces[iface], "missing iface: %s", iface)
			}

			if tt.wantAgg != nil {
				assert.NotNil(t, aggEM, "expected aggregated metrics")
				valMap := floatMetricsMap(aggEM)
				for k, v := range tt.wantAgg {
					assert.Equal(t, v, valMap[k], k)
				}
			}
		})
	}
}

func TestExportDiskUsageStats(t *testing.T) {
	tests := []struct {
		name          string
		mounts        string
		config        *configpb.ResourceUsage
		wantMounts    []string           // expected individual mount_point labels
		wantAgg       map[string]int64   // expected aggregated metric values
		wantIndiv     map[string]int64   // expected individual metric values (keyed by "mount:metric")
		wantCount     int
	}{
		{
			name:   "nil_config_aggregated_only",
			mounts: "/dev/root / ext4 rw 0 0\n/dev/sdb1 /data ext4 rw 0 0\n",
			config: nil, // defaults: aggregated=true, individual=false
			wantAgg: map[string]int64{
				"system_disk_usage_aggregated_total": 1002000, // 1000000 + 2000
				"system_disk_usage_aggregated_free":  401000,  // 400000 + 1000
			},
			wantCount: 1,
		},
		{
			name: "with_config_and_exclusions",
			mounts: `/dev/root / ext4 rw 0 0
/dev/sdb1 /data ext4 rw 0 0
proc /proc proc rw 0 0
sysfs /sys/kernel/debug sysfs rw 0 0
nsfs /run/netns/cni-123 nsfs rw 0 0
devtmpfs /dev devtmpfs rw 0 0
snap /snap/core/123 squashfs ro 0 0
`,
			config: &configpb.ResourceUsage{
				IncludeNameRegex:      proto.String("^/"),
				ExcludeNameRegex:      proto.String("^/proc"),
				ExportIndividualStats: proto.Bool(true),
				ExportAggregatedStats: proto.Bool(true),
			},
			wantMounts: []string{"/", "/data"},
			wantIndiv: map[string]int64{
				"/:system_disk_usage_total":     1000000,
				"/:system_disk_usage_free":      400000,
				"/data:system_disk_usage_total": 2000,
				"/data:system_disk_usage_free":  1000,
			},
			wantAgg: map[string]int64{
				"system_disk_usage_aggregated_total": 1002000,
				"system_disk_usage_aggregated_free":  401000,
			},
			wantCount: 3, // /, /data, aggregated
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := setupMockProcDir(t, map[string]string{"mounts": tt.mounts})
			p := newTestProbe(t, tmpDir, nil)
			if tt.config != nil {
				p.c.DiskUsageStats = tt.config
			}

			dataChan := make(chan *metrics.EventMetrics, 10)
			p.exportDiskUsageStats(time.Now(), dataChan)
			results := drainMetrics(dataChan)

			assert.Len(t, results, tt.wantCount)

			foundMounts := make(map[string]bool)
			for _, em := range results {
				mp := em.Label("mount_point")
				if mp != "" {
					foundMounts[mp] = true
					// Verify individual values
					for key, want := range tt.wantIndiv {
						parts := strings.SplitN(key, ":", 2)
						if parts[0] == mp {
							got := em.Metric(parts[1]).(*metrics.Int).Int64()
							assert.Equal(t, want, got, key)
						}
					}
					// Verify no excluded mount points leaked through
					for _, exclude := range []string{"/dev", "/sys", "/proc", "/run/netns", "/snap"} {
						if mp == exclude || strings.HasPrefix(mp, exclude+"/") {
							t.Errorf("got excluded mount point: %s", mp)
						}
					}
				} else {
					// Aggregated
					for k, want := range tt.wantAgg {
						got := em.Metric(k).(*metrics.Int).Int64()
						assert.Equal(t, want, got, k)
					}
				}
			}

			for _, mp := range tt.wantMounts {
				assert.True(t, foundMounts[mp], "missing mount: %s", mp)
			}
		})
	}
}

func TestExportDiskIOStats(t *testing.T) {
	tmpDir := setupMockProcDir(t, nil)
	p := newTestProbe(t, tmpDir, nil)

	dataChan := make(chan *metrics.EventMetrics, 10)
	p.exportDiskIOStats(time.Now(), dataChan)
	results := drainMetrics(dataChan)

	// Default config: aggregated only
	assert.Len(t, results, 1)
	em := results[0]
	assert.Equal(t, metrics.Kind(metrics.CUMULATIVE), em.Kind)
	assert.Equal(t, "", em.Label("device"))
	assert.Equal(t, 500.0*512, em.Metric("system_disk_io_aggregated_read_bytes").(*metrics.Float).Float64())
}

func TestRunOnce(t *testing.T) {
	tmpDir := setupMockProcDir(t, nil)
	p := newTestProbe(t, tmpDir, nil)
	p.name = "test_sys"

	results := p.RunOnce(context.Background())

	assert.Len(t, results, 1)
	r := results[0]
	assert.True(t, r.Success)
	assert.Equal(t, "test_sys", r.Target.Name)
	assert.True(t, r.Latency >= 0)

	// Collect all metric keys across all EventMetrics.
	allKeys := make(map[string]bool)
	for _, em := range r.Metrics {
		for _, k := range em.MetricsKeys() {
			allKeys[k] = true
		}
	}

	// Verify metrics from all subsystems are present.
	for _, key := range []string{
		// Global gauge
		"system_file_descriptors_allocated",
		"system_procs_running",
		"system_uptime_sec",
		"system_load_1m",
		"system_mem_total",
		"system_sockets_inuse",
		// Cumulative
		"system_procs_total",
		// Disk usage
		"system_disk_usage_aggregated_total",
		"system_disk_usage_aggregated_free",
		// Disk IO
		"system_disk_io_aggregated_read_bytes",
		"system_disk_io_aggregated_write_bytes",
	} {
		assert.True(t, allKeys[key], "missing metric: %s", key)
	}
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
