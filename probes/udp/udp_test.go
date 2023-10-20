// Copyright 2017-2023 The Cloudprober Authors.
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

// Workaround to skip UDP tests using a tag, until
// https://github.com/cloudprober/cloudprober/issues/199 is fixed.
//go:build !skip_udp_probe_test
// +build !skip_udp_probe_test

package udp

import (
	"context"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/common/iputils"
	"github.com/cloudprober/cloudprober/internal/sysvars"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/udp/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type serverConnStats struct {
	sync.Mutex
	msgCt map[string]int
}

func debugLog(t *testing.T, format string, args ...interface{}) {
	if os.Getenv("ACTIONS_RUNNER_DEBUG") != "true" {
		return
	}
	t.Logf(format, args...)
}

func startUDPServer(ctx context.Context, t *testing.T, drop bool, delay time.Duration) (int, *serverConnStats) {
	conn, err := net.ListenUDP("udp", nil)
	if err != nil {
		t.Fatalf("Starting UDP server failed: %v", err)
	}
	t.Logf("Recv addr: %s", conn.LocalAddr().String())
	// Simple loop to ECHO data.
	scs := &serverConnStats{
		msgCt: make(map[string]int),
	}

	go func() {
		timeout := time.Millisecond * 100
		maxLen := 1500
		b := make([]byte, maxLen)
		for {
			select {
			case <-ctx.Done():
				conn.Close()
				return
			default:
			}

			conn.SetReadDeadline(time.Now().Add(timeout))
			msgLen, addr, err := conn.ReadFromUDP(b)
			if err != nil {
				if !isClientTimeout(err) {
					t.Logf("Error receiving message: %v", err)
				}
				continue
			}
			debugLog(t, "Message from %s, size: %d", addr.String(), msgLen)
			scs.Lock()
			scs.msgCt[addr.String()]++
			scs.Unlock()
			if drop {
				continue
			}
			go func(b []byte, addr *net.UDPAddr) {
				if delay != 0 {
					time.Sleep(delay)
				}
				conn.SetWriteDeadline(time.Now().Add(timeout))
				if _, err := conn.WriteToUDP(b, addr); err != nil {
					t.Logf("Error sending message %s: %v", b, err)
				}
				debugLog(t, "Sent message to %s", addr.String())
			}(append([]byte{}, b[:msgLen]...), addr)
		}
	}()

	return conn.LocalAddr().(*net.UDPAddr).Port, scs
}

const numTxPorts = 2

func ipVersionForTest(t *testing.T, testTarget string) int {
	t.Helper()

	ips, err := net.LookupIP(testTarget)
	if err != nil {
		t.Logf("Error resolving test target: %v, defaulting to IPv4 for testing", err)
		return 4
	}

	for _, ip := range ips {
		if iputils.IPVersion(ip) == 6 {
			return 6
		}
	}
	return 4
}

func runProbe(t *testing.T, interval, timeout time.Duration, probesToSend int, scs *serverConnStats, conf *configpb.ProbeConf) *Probe {
	t.Helper()

	testTarget := "localhost"

	ctx, cancelCtx := context.WithCancel(context.Background())
	var wg sync.WaitGroup

	sysvars.Init(&logger.Logger{}, nil)
	p := &Probe{}

	conf.NumTxPorts = proto.Int32(numTxPorts)
	opts := &options.Options{
		IPVersion:           ipVersionForTest(t, testTarget),
		Targets:             targets.StaticTargets(testTarget),
		Interval:            interval,
		Timeout:             timeout,
		ProbeConf:           conf,
		StatsExportInterval: 10 * time.Second,
	}
	if err := p.Init("udp", opts); err != nil {
		t.Fatalf("Error initializing UDP probe: %v", err)
	}
	p.targets = p.opts.Targets.ListEndpoints()
	p.initProbeRunResults()

	for _, conn := range p.connList {
		wg.Add(1)
		go func(c *net.UDPConn) {
			defer wg.Done()
			p.recvLoop(ctx, c)
		}(conn)
	}

	time.Sleep(time.Second)

	wg.Add(1)
	go func() {
		defer wg.Done()

		flushTicker := time.NewTicker(p.flushIntv)
		for {
			select {
			case <-ctx.Done():
				flushTicker.Stop()
				return
			case <-flushTicker.C:
				p.processPackets()
			}
		}
	}()

	time.Sleep(interval)
	for i := 0; i < probesToSend; i++ {
		p.runProbe()
		time.Sleep(interval)
	}

	// Sleep for 2*statsExportIntv, to make sure that stats are updated and
	// exported.
	time.Sleep(2 * interval)
	time.Sleep(2 * timeout)

	scs.Lock()
	defer scs.Unlock()
	if len(scs.msgCt) != len(p.connList) {
		t.Errorf("Got packets over %d connections, required %d", len(scs.msgCt), len(p.connList))
	}
	debugLog(t, "Echo server stats: %v", scs.msgCt)

	cancelCtx()
	wg.Wait()

	return p
}

func TestSuccessMultipleCasesResultPerPort(t *testing.T) {
	cases := []struct {
		name          string
		interval      time.Duration
		timeout       time.Duration
		delay         time.Duration
		probeCount    int
		useAllPorts   bool
		pktCount      int64
		metricsByPort bool
	}{
		// 10 probes, probing each target from 2 ports, at the interval of 200ms, with 100ms timeout and 10ms delay on server.
		{"success_normal", 100, 90, 5, 10, true, 10, true},
		// 10 probes, probing each target from 2 ports, at the interval of 200ms, with 100ms timeout and 10ms delay on server.
		{"success_normal_default_metrics", 100, 90, 5, 10, true, 20, false}, // default metrics style
		// 20 probes, probing each target from 2 ports, at the interval of 100ms, with 250ms timeout and 50ms delay on server.
		{"success_timeout_larger_than_interval_1", 100, 500, 50, 10, true, 10, true},
		// 20 probes, probing each target from 2 ports, at the interval of 100ms, with 1000ms timeout and 200ms delay on server.
		{"success_timeout_larger_than_interval_2", 100, 500, 200, 10, true, 10, true},
		// 10 probes, probing each target just once, at the interval of 200ms, with 100ms timeout and 10ms delay on server.
		{"single_port", 200, 100, 10, 10, false, 5, true},
	}

	for _, c := range cases {
		t.Run("Case-"+c.name, func(t *testing.T) {
			ctx, cancelServerCtx := context.WithCancel(context.Background())
			port, scs := startUDPServer(ctx, t, false, c.delay*time.Millisecond)
			debugLog(t, "Case(%s): started server on port %d with delay %v", c.name, port, c.delay)

			conf := &configpb.ProbeConf{
				UseAllTxPortsPerProbe: proto.Bool(c.useAllPorts),
				Port:                  proto.Int32(int32(port)),
				ExportMetricsByPort:   proto.Bool(c.metricsByPort),
			}

			p := runProbe(t, c.interval*time.Millisecond, c.timeout*time.Millisecond, c.probeCount, scs, conf)
			cancelServerCtx()

			if len(p.connList) != numTxPorts {
				t.Errorf("Case(%s): len(p.connList)=%d, want %d", c.name, len(p.connList), numTxPorts)
			}

			portsList := p.srcPortList
			if !c.metricsByPort {
				portsList = []string{""}
			}

			for _, port := range portsList {
				res := p.res[flow{port, "localhost"}]
				assert.GreaterOrEqual(t, res.total, c.pktCount/2, "total")
				assert.GreaterOrEqual(t, res.success, c.pktCount/2, "success")
				assert.Equal(t, res.total-res.success, res.delayed, "delayed")
			}
		})
	}
}

func extractMetric(em *metrics.EventMetrics, key string) int64 {
	return em.Metric(key).(*metrics.Int).Int64()
}

func TestExport(t *testing.T) {
	res := probeResult{
		total:   3,
		success: 2,
		delayed: 1,
		latency: metrics.NewFloat(100.),
	}
	conf := configpb.ProbeConf{
		ExportMetricsByPort: proto.Bool(true),
		Port:                proto.Int32(1234),
	}
	m := res.eventMetrics("probe", &options.Options{}, flow{"port", "target"}, &conf)
	if r := extractMetric(m, "total-per-port"); r != 3 {
		t.Errorf("extractMetric(m,\"total-per-port\")=%d, want 3", r)
	}
	if r := extractMetric(m, "success-per-port"); r != 2 {
		t.Errorf("extractMetric(m,\"success-per-port\")=%d, want 2", r)
	}
	if got, want := m.Label("src_port"), "port"; got != want {
		t.Errorf("m.Label(\"src_port\")=%q, want %q", got, want)
	}
	if got, want := m.Label("dst_port"), "1234"; got != want {
		t.Errorf("m.Label(\"dst_port\")=%q, want %q", got, want)
	}
	conf = configpb.ProbeConf{
		ExportMetricsByPort: proto.Bool(false),
		Port:                proto.Int32(1234),
	}
	m = res.eventMetrics("probe", &options.Options{}, flow{"port", "target"}, &conf)
	if r := extractMetric(m, "total"); r != 3 {
		t.Errorf("extractMetric(m,\"total\")=%d, want 3", r)
	}
	if r := extractMetric(m, "success"); r != 2 {
		t.Errorf("extractMetric(m,\"success\")=%d, want 2", r)
	}
	if got, want := m.Label("src_port"), ""; got != want {
		t.Errorf("m.Label(\"src_port\")=%q, want %q", got, want)
	}
	if got, want := m.Label("dst_port"), ""; got != want {
		t.Errorf("m.Label(\"dst_port\")=%q, want %q", got, want)
	}
}

func TestLossAndDelayed(t *testing.T) {
	var pktCount int64 = 10
	cases := []struct {
		name     string
		drop     bool
		interval time.Duration
		timeout  time.Duration
		delay    time.Duration
		delayCt  int64
	}{
		// 10 packets, at the interval of 100ms, with 50ms timeout and drop on server.
		{"loss", true, 10, 5, 0, 0},
		// 10 packets, at the interval of 100ms, with 50ms timeout and 67ms delay on server.
		{"delayed_1", false, 20, 10, 15, pktCount},
		// 10 packets, at the interval of 100ms, with 250ms timeout and 300ms delay on server.
		{"delayed_2", false, 10, 12, 15, pktCount},
	}

	for _, c := range cases {
		t.Run("Case-"+c.name, func(t *testing.T) {
			ctx, cancelServerCtx := context.WithCancel(context.Background())
			port, scs := startUDPServer(ctx, t, c.drop, c.delay*time.Millisecond)

			debugLog(t, "Case(%s): started server on port %d with loss %v delay %v", c.name, port, c.drop, c.delay)

			conf := &configpb.ProbeConf{
				UseAllTxPortsPerProbe: proto.Bool(true),
				Port:                  proto.Int32(int32(port)),
				ExportMetricsByPort:   proto.Bool(true),
			}

			p := runProbe(t, c.interval*time.Millisecond, c.timeout*time.Millisecond, int(pktCount), scs, conf)
			cancelServerCtx()

			if len(p.connList) != numTxPorts {
				t.Errorf("Case(%s): len(p.connList)=%d, want %d", c.name, len(p.connList), numTxPorts)
			}

			for _, port := range p.srcPortList {
				res := p.res[flow{port, "localhost"}]
				assert.Equal(t, int64(0), res.success, "success")
				assert.GreaterOrEqual(t, res.total, pktCount/5, "total")
				assert.GreaterOrEqual(t, res.delayed, c.delayCt/5, "delayed")
			}
		})
	}
}
