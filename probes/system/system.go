// Copyright 2023 The Cloudprober Authors.
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
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/system/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
)

// Probe holds aggregate information about the probe.
type Probe struct {
	name   string
	c      *configpb.ProbeConf
	l      *logger.Logger
	opts   *options.Options
	sysDir string // For testing
}

// Init initializes the probe with the given params.
func (p *Probe) Init(name string, opts *options.Options) error {
	c, ok := opts.ProbeConf.(*configpb.ProbeConf)
	if !ok {
		return fmt.Errorf("not a system probe config")
	}

	p.name = name
	p.c = c
	p.l = opts.Logger
	p.opts = opts
	p.sysDir = "/proc"
	return nil
}

// Start starts and runs the probe indefinitely.
func (p *Probe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	ticker := time.NewTicker(p.opts.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case ts := <-ticker.C:
			// Global metrics
			em := metrics.NewEventMetrics(ts).
				AddLabel("probe", p.name).
				AddLabel("ptype", "system")

			p.exportGlobalMetrics(em)
			p.opts.RecordMetrics(endpoint.Endpoint{Name: p.name}, em, dataChan)

			// Per-interface metrics
			if p.c.GetExportNetDevStats() {
				p.exportNetDevStats(ts, dataChan)
			}
		}
	}
}

func (p *Probe) exportGlobalMetrics(em *metrics.EventMetrics) {
	if p.c.GetExportFileDescriptors() {
		if err := p.addFileDescMetrics(em); err != nil {
			p.l.Warningf("Error getting file descriptor metrics: %v", err)
		}
	}
	if p.c.GetExportProcStats() {
		if err := p.addProcStats(em); err != nil {
			p.l.Warningf("Error getting proc stats: %v", err)
		}
	}
	if p.c.GetExportSockStats() {
		if err := p.addSockStats(em); err != nil {
			p.l.Warningf("Error getting sock stats: %v", err)
		}
	}
	if p.c.GetExportUptime() {
		if err := p.addUptime(em); err != nil {
			p.l.Warningf("Error getting uptime: %v", err)
		}
	}
	if p.c.GetExportLoadAvg() {
		if err := p.addLoadAvg(em); err != nil {
			p.l.Warningf("Error getting load average: %v", err)
		}
	}
}

func parseValue(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func (p *Probe) exportNetDevStats(ts time.Time, dataChan chan *metrics.EventMetrics) {
	f, err := os.Open(filepath.Join(p.sysDir, "net/dev"))
	if err != nil {
		p.l.Warningf("Error getting net dev stats: %v", err)
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	// Skip header lines
	if scanner.Scan(); !scanner.Scan() {
		return
	}

	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Split(line, ":")
		if len(parts) != 2 {
			continue
		}
		iface := strings.TrimSpace(parts[0])

		fields := strings.Fields(parts[1])
		if len(fields) < 16 {
			continue
		}

		em := metrics.NewEventMetrics(ts).
			AddLabel("probe", p.name).
			AddLabel("ptype", "system").
			AddLabel("iface", iface)

		// Standard /proc/net/dev columns
		if v, err := parseValue(fields[0]); err == nil {
			em.AddMetric("system_net_rx_bytes", metrics.NewFloat(v))
		}
		if v, err := parseValue(fields[1]); err == nil {
			em.AddMetric("system_net_rx_packets", metrics.NewFloat(v))
		}
		if v, err := parseValue(fields[2]); err == nil {
			em.AddMetric("system_net_rx_errors", metrics.NewFloat(v))
		}
		if v, err := parseValue(fields[3]); err == nil {
			em.AddMetric("system_net_rx_dropped", metrics.NewFloat(v))
		}

		if v, err := parseValue(fields[8]); err == nil {
			em.AddMetric("system_net_tx_bytes", metrics.NewFloat(v))
		}
		if v, err := parseValue(fields[9]); err == nil {
			em.AddMetric("system_net_tx_packets", metrics.NewFloat(v))
		}
		if v, err := parseValue(fields[10]); err == nil {
			em.AddMetric("system_net_tx_errors", metrics.NewFloat(v))
		}
		if v, err := parseValue(fields[11]); err == nil {
			em.AddMetric("system_net_tx_dropped", metrics.NewFloat(v))
		}

		p.opts.RecordMetrics(endpoint.Endpoint{Name: p.name}, em, dataChan)
	}
	if err := scanner.Err(); err != nil {
		p.l.Warningf("Error reading net dev stats: %v", err)
	}
}

func (p *Probe) addFileDescMetrics(em *metrics.EventMetrics) error {
	b, err := os.ReadFile(filepath.Join(p.sysDir, "sys/fs/file-nr"))
	if err != nil {
		return err
	}
	fields := strings.Fields(string(b))
	if len(fields) < 3 {
		return fmt.Errorf("unexpected format in file-nr: %s", string(b))
	}

	alloc, err := parseValue(fields[0])
	if err != nil {
		return err
	}
	max, err := parseValue(fields[2])
	if err != nil {
		return err
	}

	em.AddMetric("system_file_descriptors_allocated", metrics.NewFloat(alloc))
	em.AddMetric("system_file_descriptors_max", metrics.NewFloat(max))
	return nil
}

func (p *Probe) addProcStats(em *metrics.EventMetrics) error {
	f, err := os.Open(filepath.Join(p.sysDir, "stat"))
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}

		switch fields[0] {
		case "procs_running":
			v, _ := parseValue(fields[1])
			em.AddMetric("system_procs_running", metrics.NewFloat(v))
		case "procs_blocked":
			v, _ := parseValue(fields[1])
			em.AddMetric("system_procs_blocked", metrics.NewFloat(v))
		case "processes":
			v, _ := parseValue(fields[1])
			em.AddMetric("system_procs_total", metrics.NewFloat(v))
		}
	}
	return scanner.Err()
}

func (p *Probe) addSockStats(em *metrics.EventMetrics) error {
	f, err := os.Open(filepath.Join(p.sysDir, "net/sockstat"))
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		// Example: "TCP: inuse 1 orphan 0 tw 0 alloc 1 mem 1"
		protocol := strings.TrimSuffix(fields[0], ":")

		for i := 1; i < len(fields)-1; i += 2 {
			key := fields[i]
			val, err := parseValue(fields[i+1])
			if err != nil {
				continue
			}
			metricName := fmt.Sprintf("system_sockets_%s_%s", strings.ToLower(protocol), key)
			em.AddMetric(metricName, metrics.NewFloat(val))
		}

		// Also parse sockets: used X
		if fields[0] == "sockets:" && len(fields) >= 3 && fields[1] == "used" {
			val, _ := parseValue(fields[2])
			em.AddMetric("system_sockets_in_use", metrics.NewFloat(val))
		}
	}
	return scanner.Err()
}

func (p *Probe) addUptime(em *metrics.EventMetrics) error {
	b, err := os.ReadFile(filepath.Join(p.sysDir, "uptime"))
	if err != nil {
		return err
	}
	fields := strings.Fields(string(b))
	if len(fields) < 1 {
		return fmt.Errorf("unexpected format in uptime")
	}

	val, err := parseValue(fields[0])
	if err != nil {
		return err
	}
	em.AddMetric("system_uptime_sec", metrics.NewFloat(val))
	return nil
}

func (p *Probe) addLoadAvg(em *metrics.EventMetrics) error {
	b, err := os.ReadFile(filepath.Join(p.sysDir, "loadavg"))
	if err != nil {
		return err
	}
	fields := strings.Fields(string(b))
	if len(fields) < 3 {
		return fmt.Errorf("unexpected format in loadavg")
	}

	if v, err := parseValue(fields[0]); err == nil {
		em.AddMetric("system_load_1m", metrics.NewFloat(v))
	}
	if v, err := parseValue(fields[1]); err == nil {
		em.AddMetric("system_load_5m", metrics.NewFloat(v))
	}
	if v, err := parseValue(fields[2]); err == nil {
		em.AddMetric("system_load_15m", metrics.NewFloat(v))
	}
	return nil
}
