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
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
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

	// For testing
	diskUsageFunc func(path string) (uint64, uint64, error)
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
	p.diskUsageFunc = diskUsage
	return nil
}

func diskUsage(path string) (uint64, uint64, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, err
	}
	// Available blocks * size per block = available bytes
	free := stat.Bavail * uint64(stat.Bsize)
	total := stat.Blocks * uint64(stat.Bsize)
	return total, free, nil
}

func (p *Probe) exportGlobalMetrics(em, emCum *metrics.EventMetrics) {
	if !p.c.GetDisableFileDescriptors() {
		if err := p.addFileDescMetrics(em); err != nil {
			p.l.Warningf("Error getting file descriptor metrics: %v", err)
		}
	}
	if !p.c.GetDisableProcStats() {
		if err := p.addProcStats(em, emCum); err != nil {
			p.l.Warningf("Error getting proc stats: %v", err)
		}
	}
	if !p.c.GetDisableSockStats() {
		if err := p.addSockStats(em); err != nil {
			p.l.Warningf("Error getting sock stats: %v", err)
		}
	}
	if !p.c.GetDisableUptime() {
		if err := p.addUptime(em); err != nil {
			p.l.Warningf("Error getting uptime: %v", err)
		}
	}
	if !p.c.GetDisableLoadAvg() {
		if err := p.addLoadAvg(em); err != nil {
			p.l.Warningf("Error getting load average: %v", err)
		}
	}
	if !p.c.GetDisableMemoryUsage() {
		if err := p.addMemStats(em); err != nil {
			p.l.Warningf("Error getting memory stats: %v", err)
		}
	}
}

func parseValue(s string) (float64, error) {
	return strconv.ParseFloat(s, 64)
}

func matchDevice(name, include, exclude string) bool {
	if include != "" {
		matched, err := regexp.MatchString(include, name)
		if err != nil || !matched {
			return false
		}
	}
	if exclude != "" {
		matched, err := regexp.MatchString(exclude, name)
		if err == nil && matched {
			return false
		}
	}
	return true
}

func (p *Probe) exportNetDevStats(ts time.Time, dataChan chan *metrics.EventMetrics) {
	if p.c.GetNetDevStats().GetDisabled() {
		return
	}

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

	// Parse net dev stats
	var rxBytes, txBytes, rxPackets, txPackets, rxErrors, txErrors, rxDropped, txDropped float64

	// Per-device export
	recordMetrics := func(iface string, vRxBytes, vRxPackets, vRxErrors, vRxDropped, vTxBytes, vTxPackets, vTxErrors, vTxDropped float64, agg bool) {
		em := metrics.NewEventMetrics(ts).
			AddLabel("probe", p.name).
			AddLabel("ptype", "system")
		em.Kind = metrics.CUMULATIVE

		if iface != "" {
			em.AddLabel("iface", iface)
		}

		modifier := ""
		if agg {
			modifier = "aggregated_"
		}

		em.AddMetric("system_net_"+modifier+"rx_bytes", metrics.NewFloat(vRxBytes))
		em.AddMetric("system_net_"+modifier+"rx_packets", metrics.NewFloat(vRxPackets))
		em.AddMetric("system_net_"+modifier+"rx_errors", metrics.NewFloat(vRxErrors))
		em.AddMetric("system_net_"+modifier+"rx_dropped", metrics.NewFloat(vRxDropped))
		em.AddMetric("system_net_"+modifier+"tx_bytes", metrics.NewFloat(vTxBytes))
		em.AddMetric("system_net_"+modifier+"tx_packets", metrics.NewFloat(vTxPackets))
		em.AddMetric("system_net_"+modifier+"tx_errors", metrics.NewFloat(vTxErrors))
		em.AddMetric("system_net_"+modifier+"tx_dropped", metrics.NewFloat(vTxDropped))

		p.opts.RecordMetrics(endpoint.Endpoint{Name: p.name}, em, dataChan)
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

		// Values
		vRxBytes, _ := parseValue(fields[0])
		vRxPackets, _ := parseValue(fields[1])
		vRxErrors, _ := parseValue(fields[2])
		vRxDropped, _ := parseValue(fields[3])
		vTxBytes, _ := parseValue(fields[8])
		vTxPackets, _ := parseValue(fields[9])
		vTxErrors, _ := parseValue(fields[10])
		vTxDropped, _ := parseValue(fields[11])

		// Aggregation
		rxBytes += vRxBytes
		rxPackets += vRxPackets
		rxErrors += vRxErrors
		rxDropped += vRxDropped
		txBytes += vTxBytes
		txPackets += vTxPackets
		txErrors += vTxErrors
		txDropped += vTxDropped

		if p.c.GetNetDevStats().GetExportIndividualStats() {
			if matchDevice(iface, p.c.GetNetDevStats().GetIncludeNameRegex(), p.c.GetNetDevStats().GetExcludeNameRegex()) {
				recordMetrics(iface, vRxBytes, vRxPackets, vRxErrors, vRxDropped, vTxBytes, vTxPackets, vTxErrors, vTxDropped, false)
			}
		}
	}

	if p.c.GetNetDevStats().GetExportAggregatedStats() {
		recordMetrics("", rxBytes, rxPackets, rxErrors, rxDropped, txBytes, txPackets, txErrors, txDropped, true)
	}

	if err := scanner.Err(); err != nil {
		p.l.Warningf("Error reading net dev stats: %v", err)
	}
}

func (p *Probe) getMountPoints() ([]string, error) {
	f, err := os.Open(filepath.Join(p.sysDir, "mounts"))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var mounts []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) >= 2 {
			mounts = append(mounts, fields[1])
		}
	}
	return mounts, scanner.Err()
}

func (p *Probe) exportDiskUsageStats(ts time.Time, dataChan chan *metrics.EventMetrics) {
	if p.diskUsageFunc == nil {
		return
	}

	config := p.c.GetDiskUsageStats()
	var mounts []string

	if config != nil {
		if config.GetDisabled() {
			return
		}
		var err error
		mounts, err = p.getMountPoints()
		if err != nil {
			p.l.Warningf("Error reading mounts: %v", err)
			// Fallback to / if we can't read mounts
			mounts = []string{"/"}
		}
	} else {
		// Currently checking only root
		mounts = []string{"/"}
	}

	var aggTotal, aggFree, aggUsed uint64

	recordMetrics := func(mount string, free, used uint64, agg bool) {
		em := metrics.NewEventMetrics(ts).
			AddLabel("probe", p.name).
			AddLabel("ptype", "system")
		if mount != "" {
			em.AddLabel("mount_point", mount)
		}
		em.Kind = metrics.GAUGE

		modifier := ""
		if agg {
			modifier = "aggregated_"
		}
		em.AddMetric("system_disk_"+modifier+"total", metrics.NewInt(int64(free+used)))
		em.AddMetric("system_disk_"+modifier+"free", metrics.NewInt(int64(free)))
		p.opts.RecordMetrics(endpoint.Endpoint{Name: p.name}, em, dataChan)
	}

	for _, mount := range mounts {
		excluded := false
		for _, exclude := range []string{"/dev", "/sys", "/proc", "/run/netns", "/snap"} {
			if mount == exclude || strings.HasPrefix(mount, exclude+"/") {
				excluded = true
				break
			}
		}
		if excluded {
			continue
		}

		if config != nil {
			if !matchDevice(mount, config.GetIncludeNameRegex(), config.GetExcludeNameRegex()) {
				continue
			}
		}

		total, free, err := p.diskUsageFunc(mount)
		if err != nil {
			p.l.Warningf("Error getting disk usage for %s: %v", mount, err)
			continue
		}
		used := total - free

		aggTotal += total
		aggFree += free
		aggUsed += used

		exportIndividual := true
		if config != nil {
			exportIndividual = config.GetExportIndividualStats()
		}

		if exportIndividual {
			recordMetrics(mount, free, used, false)
		}
	}

	if config != nil && config.GetExportAggregatedStats() {
		recordMetrics("", aggFree, aggUsed, true)
	}
}

func (p *Probe) exportDiskIOStats(ts time.Time, dataChan chan *metrics.EventMetrics) {
	if p.c.GetDiskIoStats().GetDisabled() {
		return
	}

	f, err := os.Open(filepath.Join(p.sysDir, "diskstats"))
	if err != nil {
		p.l.Warningf("Error getting disk IO stats: %v", err)
		return
	}
	defer f.Close()

	var readBytes, writeBytes, readCount, writeCount float64

	// Per-device export
	recordMetrics := func(device string, rBytes, wBytes, rCount, wCount float64, agg bool) {
		em := metrics.NewEventMetrics(ts).
			AddLabel("probe", p.name).
			AddLabel("ptype", "system")
		if device != "" {
			em.AddLabel("device", device)
		}
		em.Kind = metrics.CUMULATIVE

		modifier := ""
		if agg {
			modifier = "aggregated_"
		}

		em.AddMetric("system_disk_"+modifier+"read_bytes", metrics.NewFloat(rBytes))
		em.AddMetric("system_disk_"+modifier+"write_bytes", metrics.NewFloat(wBytes))
		em.AddMetric("system_disk_"+modifier+"read_count", metrics.NewFloat(rCount))
		em.AddMetric("system_disk_"+modifier+"write_count", metrics.NewFloat(wCount))

		p.opts.RecordMetrics(endpoint.Endpoint{Name: p.name}, em, dataChan)
	}

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		// Linux 2.6+ /proc/diskstats usually has 14+ fields
		// 8 0 sda 100 200 300 400 ...
		if len(fields) < 14 {
			continue
		}
		device := fields[2]
		// Filter out loopback and ram devices
		if strings.HasPrefix(device, "loop") || strings.HasPrefix(device, "ram") || strings.HasPrefix(device, "sr") {
			continue
		}

		readsCompleted, _ := parseValue(fields[3])
		sectorsRead, _ := parseValue(fields[5]) // 512 bytes per sector usually
		writeCompleted, _ := parseValue(fields[7])
		sectorsWritten, _ := parseValue(fields[9])

		// bytes = sectors * 512
		vReadBytes := sectorsRead * 512
		vWriteBytes := sectorsWritten * 512

		// Aggregation
		readBytes += vReadBytes
		writeBytes += vWriteBytes
		readCount += readsCompleted
		writeCount += writeCompleted

		if p.c.GetDiskIoStats().GetExportIndividualStats() {
			if matchDevice(device, p.c.GetDiskIoStats().GetIncludeNameRegex(), p.c.GetDiskIoStats().GetExcludeNameRegex()) {
				recordMetrics(device, vReadBytes, vWriteBytes, readsCompleted, writeCompleted, false)
			}
		}
	}

	if p.c.GetDiskIoStats().GetExportAggregatedStats() {
		recordMetrics("", readBytes, writeBytes, readCount, writeCount, true)
	}

	if err := scanner.Err(); err != nil {
		p.l.Warningf("Error reading diskstats: %v", err)
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

func (p *Probe) addProcStats(em, emCum *metrics.EventMetrics) error {
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
			emCum.AddMetric("system_procs_total", metrics.NewFloat(v))
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

func (p *Probe) addMemStats(em *metrics.EventMetrics) error {
	f, err := os.Open(filepath.Join(p.sysDir, "meminfo"))
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
		// Example: MemTotal:        16393932 kB
		key := strings.TrimSuffix(fields[0], ":")
		valStr := fields[1]
		val, err := parseValue(valStr)
		if err != nil {
			continue
		}
		// Convert kB to Bytes if unit is kB
		if len(fields) >= 3 && fields[2] == "kB" {
			val = val * 1024
		}

		switch key {
		case "MemTotal":
			em.AddMetric("system_mem_total", metrics.NewFloat(val))
		case "MemFree":
			em.AddMetric("system_mem_free", metrics.NewFloat(val))
		case "MemAvailable":
			em.AddMetric("system_mem_available", metrics.NewFloat(val))
		case "Buffers":
			em.AddMetric("system_mem_buffers", metrics.NewFloat(val))
		case "Cached":
			em.AddMetric("system_mem_cached", metrics.NewFloat(val))
		}
	}
	return scanner.Err()
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
			// Gauge metrics
			em := metrics.NewEventMetrics(ts).
				AddLabel("probe", p.name).
				AddLabel("ptype", "system")
			em.Kind = metrics.GAUGE

			// Cumulative metrics
			emCum := metrics.NewEventMetrics(ts).
				AddLabel("probe", p.name).
				AddLabel("ptype", "system")
			emCum.Kind = metrics.CUMULATIVE

			p.exportGlobalMetrics(em, emCum)
			p.opts.RecordMetrics(endpoint.Endpoint{Name: p.name}, em, dataChan)
			p.opts.RecordMetrics(endpoint.Endpoint{Name: p.name}, emCum, dataChan)

			// Per-interface metrics
			if !p.c.GetNetDevStats().GetDisabled() {
				p.exportNetDevStats(ts, dataChan)
			}
			if !p.c.GetDiskIoStats().GetDisabled() {
				p.exportDiskIOStats(ts, dataChan)
			}
			p.exportDiskUsageStats(ts, dataChan)
		}
	}
}
