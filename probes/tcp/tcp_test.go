// Copyright 2022 The Cloudprober Authors.
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

package tcp

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/tcp/proto"
	"github.com/cloudprober/cloudprober/targets/endpoint"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

type dialState struct {
	network, address string
}

func testDialContext(ds *dialState) func(ctx context.Context, network, addr string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		ds.network = network
		ds.address = addr

		if strings.Contains(addr, ":82") {
			return nil, fmt.Errorf("unsupported port")
		}
		return nil, nil
	}
}

func TestRunProbe(t *testing.T) {
	tests := []struct {
		desc, addr             string
		ipVersion              int
		wantNetwork, wantAddr  string
		wantSuccess, wantTotal int64
	}{
		{
			desc:        "success-default-ip",
			addr:        "test.com:80",
			wantNetwork: "tcp",
			wantAddr:    "test.com:80",
			wantSuccess: 1,
			wantTotal:   1,
		},
		{
			desc:        "success-ipv4",
			addr:        "test.com:81",
			ipVersion:   4,
			wantNetwork: "tcp4",
			wantAddr:    "test.com:81",
			wantSuccess: 1,
			wantTotal:   1,
		},
		{
			desc:        "fail-bad-port-ipv6",
			addr:        "test.com:82",
			ipVersion:   6,
			wantNetwork: "tcp6",
			wantAddr:    "test.com:82",
			wantSuccess: 0,
			wantTotal:   1,
		},
	}
	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			p := &Probe{}
			opts := options.DefaultOptions()
			opts.IPVersion = test.ipVersion

			err := p.Init("test-probe", opts)
			if err != nil {
				t.Errorf("error initializing probe: %v", err)
			}

			host, portStr, _ := net.SplitHostPort(test.addr)
			port, _ := strconv.Atoi(portStr)

			ds := &dialState{}
			p.dialContext = testDialContext(ds)

			runReq := &sched.RunProbeForTargetRequest{Target: endpoint.Endpoint{Name: host, Port: port}}
			p.runProbe(context.Background(), runReq)

			if ds.network != test.wantNetwork {
				t.Errorf("Got network: %s, wanted: %s", ds.network, test.wantNetwork)
			}
			if ds.address != test.wantAddr {
				t.Errorf("Got address: %s, wanted: %s", ds.address, test.wantAddr)
			}

			result := runReq.Result.(*probeResult)
			if result.total != test.wantTotal {
				t.Errorf("Got total: %d, wanted: %d", result.total, test.wantTotal)
			}
			if result.success != test.wantSuccess {
				t.Errorf("Got success: %d, wanted: %d", result.success, test.wantSuccess)
			}
		})
	}

}

func TestConnectAndHandshake(t *testing.T) {
	tests := []struct {
		desc                   string
		addr                   string
		tlsHandshakeEnabled    bool
		tlsConfig              *tls.Config
		dialError              error
		wantSuccess            bool
		wantNonZeroConnLatency bool
		wantNonZeroTLSLatency  bool
	}{
		{
			desc:                   "success-with-tls-handshake",
			addr:                   "test.com:443",
			tlsHandshakeEnabled:    true,
			wantSuccess:            true,
			wantNonZeroConnLatency: true,
			wantNonZeroTLSLatency:  true,
		},
		{
			desc:                "success-without-tls-handshake",
			addr:                "test.com:80",
			tlsHandshakeEnabled: false,
			wantSuccess:         true,
		},
		{
			desc:                "dial-error",
			addr:                "invalid.com:80",
			tlsHandshakeEnabled: true,
			dialError:           fmt.Errorf("dial error"),
			wantSuccess:         false,
		},
		{
			desc:                   "handshake-error",
			addr:                   "test.com:443",
			tlsHandshakeEnabled:    true,
			tlsConfig:              &tls.Config{ServerName: "error.com"},
			wantSuccess:            false,
			wantNonZeroConnLatency: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			p := &Probe{
				c:         &configpb.ProbeConf{},
				opts:      options.DefaultOptions(),
				network:   "tcp",
				tlsConfig: test.tlsConfig,
			}

			// Extract host from addr
			host, _, _ := net.SplitHostPort(test.addr)

			if test.tlsHandshakeEnabled {
				p.c.TlsHandshake = proto.Bool(true)
			}

			p.dialContext = func(ctx context.Context, network, addr string) (net.Conn, error) {
				time.Sleep(1 * time.Millisecond)
				if addr == test.addr && test.dialError == nil {
					return &net.TCPConn{}, nil
				}
				return nil, test.dialError
			}

			p.handshakeContext = func(ctx context.Context, _ net.Conn, tlsConfig *tls.Config) error {
				if tlsConfig.ServerName == "error.com" {
					return fmt.Errorf("handshake error")
				}
				assert.Equal(t, host, tlsConfig.ServerName)
				time.Sleep(1 * time.Millisecond)
				return nil
			}

			result := &probeResult{
				connLatency:         metrics.NewFloat(0),
				tlsHandshakeLatency: metrics.NewFloat(0),
			}

			err := p.connectAndHandshake(context.Background(), test.addr, host, result)

			if test.wantSuccess {
				if err != nil {
					t.Errorf("Expected success, but got error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error, but got success")
				}
			}

			if test.wantNonZeroConnLatency {
				assert.NotZero(t, result.connLatency.(*metrics.Float).Float64(), "Expected non-zero connection latency")
			} else {
				assert.Zero(t, result.connLatency.(*metrics.Float).Float64(), "Expected zero connection latency")
			}
			if test.wantNonZeroTLSLatency {
				assert.NotZero(t, result.tlsHandshakeLatency.(*metrics.Float).Float64(), "Expected non-zero TLS handshake latency")
			} else {
				assert.Zero(t, result.tlsHandshakeLatency.(*metrics.Float).Float64(), "Expected zero TLS handshake latency")
			}
		})
	}
}

func TestNewResultAndMetrics(t *testing.T) {
	total, success := int64(10), int64(8)
	connLatency := metrics.NewFloat(0.1)
	tlsHandshakeLatency := metrics.NewFloat(0.2)

	tests := []struct {
		desc                    string
		enableTlsHandshake      bool
		wantMetrics             map[string]int64
		wantConnLatency         float64
		wantTLSHandshakeLatency float64
	}{
		{
			desc: "basic-metrics",
			wantMetrics: map[string]int64{
				"total":   total,
				"success": success,
			},
			wantConnLatency:         -1,
			wantTLSHandshakeLatency: -1,
		},
		{
			desc:               "metrics-with-tls-handshake",
			enableTlsHandshake: true,
			wantMetrics: map[string]int64{
				"total":   total,
				"success": success,
			},
			wantConnLatency:         0.1,
			wantTLSHandshakeLatency: 0.2,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ts := time.Now()

			p := &Probe{
				opts: options.DefaultOptions(),
			}
			p.c = &configpb.ProbeConf{}
			if test.enableTlsHandshake {
				p.c.TlsHandshake = proto.Bool(true)
			}

			result := p.newResult().(*probeResult)
			result.total = total
			result.success = success
			// Verify that connLatency and tlsHandshakeLatency are initialized
			if test.enableTlsHandshake {
				assert.NotNil(t, result.connLatency)
				assert.NotNil(t, result.tlsHandshakeLatency)
				result.connLatency = connLatency
				result.tlsHandshakeLatency = tlsHandshakeLatency
			}

			emList := result.Metrics(ts, 0, &options.Options{})
			if len(emList) != 1 {
				t.Fatalf("Expected 1 EventMetrics, got %d", len(emList))
			}

			em := emList[0]

			// Verify metrics
			for metricName, wantValue := range test.wantMetrics {
				assert.Equal(t, wantValue, em.Metric(metricName).(*metrics.Int).Int64())
			}

			// Verify label
			assert.Equal(t, "tcp", em.Label("ptype"))

			// Verify conn_latency and tls_handshake_latency if applicable
			if test.wantConnLatency == -1 {
				assert.Nil(t, em.Metric("connect_latency"))
			} else {
				assert.Equal(t, test.wantConnLatency, em.Metric("connect_latency").(*metrics.Float).Float64())
			}

			if test.wantTLSHandshakeLatency == -1 {
				assert.Nil(t, em.Metric("tls_handshake_latency"))
			} else {
				assert.Equal(t, test.wantTLSHandshakeLatency, em.Metric("tls_handshake_latency").(*metrics.Float).Float64())
			}
		})
	}
}
