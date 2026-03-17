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

package dns

import (
	"context"
	"runtime"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/dns/proto"
	"github.com/cloudprober/cloudprober/probes/options"
	"github.com/cloudprober/cloudprober/targets"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestRunOnce(t *testing.T) {
	tests := []struct {
		name             string
		target           string
		queryType        configpb.QueryType
		resolvedDomain   string
		requestsPerProbe int32
		wantSuccess      bool
		wantNil          bool
	}{
		{
			name:           "success",
			target:         "8.8.8.8",
			queryType:      configpb.QueryType_A,
			resolvedDomain: "test.com",
			wantSuccess:    true,
		},
		{
			name:           "bad_domain",
			target:         "8.8.8.8",
			queryType:      configpb.QueryType_A,
			resolvedDomain: questionBadDomain,
			wantSuccess:    false,
		},
		{
			name:           "bad_target",
			target:         "1.2.3.4",
			queryType:      configpb.QueryType_A,
			resolvedDomain: "test.com",
			wantSuccess:    false,
		},
		{
			name:             "requests_per_probe_gt_1",
			target:           "8.8.8.8",
			queryType:        configpb.QueryType_A,
			resolvedDomain:   "test.com",
			requestsPerProbe: 2,
			wantNil:          true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			probeConf := &configpb.ProbeConf{
				QueryType:      tt.queryType.Enum(),
				ResolvedDomain: proto.String(tt.resolvedDomain),
			}
			if tt.requestsPerProbe > 0 {
				probeConf.RequestsPerProbe = proto.Int32(tt.requestsPerProbe)
			}

			opts := options.DefaultOptions()
			opts.Targets = targets.StaticTargets(tt.target)
			opts.Timeout = 2 * time.Second
			opts.ProbeConf = probeConf
			opts.Logger = &logger.Logger{}
			opts.LatencyUnit = time.Millisecond

			p := &Probe{}
			if err := p.Init("dns-runonce", opts); err != nil {
				t.Fatalf("Init error: %v", err)
			}
			p.client = new(mockClient)

			results := p.RunOnce(context.Background())

			if tt.wantNil {
				assert.Nil(t, results, "expected nil results for unsupported config")
				return
			}

			assert.Equal(t, 1, len(results), "expected 1 result")
			r := results[0]
			assert.Equal(t, tt.wantSuccess, r.Success, "success mismatch")

			if tt.wantSuccess {
				// Skip latency check on Windows: timer resolution is too
				// coarse for the mock client, so time.Since can return 0.
				if runtime.GOOS != "windows" {
					assert.True(t, r.Latency > 0, "expected non-zero latency")
				}
				assert.Nil(t, r.Error, "expected no error")
				assert.NotNil(t, r.Metrics, "expected metrics")
			} else {
				assert.NotNil(t, r.Error, "expected error for failed probe")
			}
		})
	}
}
