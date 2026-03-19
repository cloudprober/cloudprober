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

package tcp

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/tcp/proto"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestRunOnce(t *testing.T) {
	// Start a TCP listener for success tests.
	ln, err := net.Listen("tcp", "localhost:0")
	if err != nil {
		t.Fatalf("Failed to start TCP listener: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := ln.Addr().(*net.TCPAddr)

	tests := []struct {
		name        string
		target      string
		port        int32
		wantSuccess bool
	}{
		{
			name:        "success",
			target:      addr.IP.String(),
			port:        int32(addr.Port),
			wantSuccess: true,
		},
		{
			name:        "connect_failure",
			target:      "localhost",
			port:        1, // unlikely to be listening
			wantSuccess: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := options.DefaultOptions()
			opts.Targets = targets.StaticTargets(tt.target)
			opts.Timeout = 2 * time.Second
			opts.ProbeConf = &configpb.ProbeConf{
				Port: proto.Int32(tt.port),
			}
			opts.Logger = &logger.Logger{}
			opts.LatencyUnit = time.Millisecond

			p := &Probe{}
			if err := p.Init("tcp-runonce", opts); err != nil {
				t.Fatalf("Init error: %v", err)
			}

			results := p.RunOnce(context.Background())

			assert.Equal(t, 1, len(results), "expected 1 result")
			r := results[0]
			assert.Equal(t, tt.wantSuccess, r.Success, "success mismatch")

			if tt.wantSuccess {
				assert.True(t, r.Latency > 0, "expected non-zero latency")
				assert.Nil(t, r.Error, "expected no error")
				assert.NotNil(t, r.Metrics, "expected metrics")
			} else {
				assert.NotNil(t, r.Error, "expected error for failed probe")
			}
		})
	}
}
