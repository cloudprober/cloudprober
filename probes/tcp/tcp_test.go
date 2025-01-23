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
	"fmt"
	"net"
	"strconv"
	"strings"
	"testing"

	"github.com/cloudprober/cloudprober/probes/common/sched"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets/endpoint"
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
