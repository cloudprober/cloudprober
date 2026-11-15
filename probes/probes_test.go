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

// This file is for external tests of the probe packages, created to provide
// tests for the probe extensions.

package probes_test

import (
	"context"
	"testing"

	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/proto"
	testdatapb "github.com/cloudprober/cloudprober/probes/testdata"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

var testProbeIntialized int

type testProbe struct{}

func (p *testProbe) Init(name string, opts *options.Options) error {
	testProbeIntialized++
	return nil
}

func (p *testProbe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {}

func TestGetExtensionProbe(t *testing.T) {
	probeDef := &configpb.ProbeDef{
		Name: proto.String("ext-probe"),
		Type: configpb.ProbeDef_EXTENSION.Enum(),
		Targets: &targetspb.TargetsDef{
			Type: &targetspb.TargetsDef_DummyTargets{},
		},
	}

	// This has the same effect as using the following in your config:
	// probe {
	//    name: "ext-probe"
	//    targets: ...
	//    ...
	//    [cloudprober.probes.testdata.fancy_probe] {
	//      name: "fancy"
	//    }
	// }
	proto.SetExtension(probeDef, testdatapb.E_FancyProbe, &testdatapb.FancyProbe{Name: proto.String("fancy")})
	probeInfo, err := probes.CreateProbe(probeDef, &options.Options{})
	if err == nil {
		t.Errorf("Expected error in building probe from extensions, got probe: %v", probeInfo)
	}
	t.Log(err.Error())

	// Register our test probe type and try again.
	probes.RegisterProbeType(200, func() probes.Probe {
		return &testProbe{}
	})

	probeInfo, err = probes.CreateProbe(probeDef, &options.Options{})
	if err != nil {
		t.Errorf("Got error in building probe from extensions: %v", err)
	}
	if probeInfo == nil || probeInfo.Name != "ext-probe" {
		t.Errorf("Extension probe not in the probes map")
	}
	_, ok := probeInfo.Probe.(*testProbe)
	if !ok {
		t.Errorf("Extension probe (%v) is not of type *testProbe", probeInfo)
	}
	if testProbeIntialized != 1 {
		t.Errorf("Extensions probe's Init() called %d times, should be called exactly once.", testProbeIntialized)
	}
}

func TestProbeInfoLatencyDistLB(t *testing.T) {
	probeDef := &configpb.ProbeDef{
		Name: proto.String("tcp-probe"),
		Type: configpb.ProbeDef_TCP.Enum(),
		Targets: &targetspb.TargetsDef{
			Type: &targetspb.TargetsDef_DummyTargets{},
		},
	}

	nativeDist, err := metrics.NewNativeDistribution(3)
	if err != nil {
		t.Fatalf("Error creating native distribution: %v", err)
	}

	for want, dist := range map[string]*metrics.Distribution{
		"":                 nil,
		"[-Inf 1 2]":       metrics.NewDistribution([]float64{1, 2}),
		"native(schema=3)": nativeDist,
	} {
		opts := options.DefaultOptions()
		opts.LatencyDist = dist
		probeInfo, err := probes.CreateProbe(probeDef, opts)
		if err != nil {
			t.Fatalf("Error creating probe: %v", err)
		}
		if probeInfo.LatencyDistLB != want {
			t.Errorf("LatencyDistLB=%q, want=%q", probeInfo.LatencyDistLB, want)
		}
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name     string
		probe    string // ProbeDef in text format, without name and targets.
		wantErr  string
		notInErr string
	}{
		{
			name:  "http_valid",
			probe: `type: HTTP http_probe { relative_url: "/health" proxy_url: "http://proxy:3128" }`,
		},
		{
			name:    "http_bad_relative_url",
			probe:   `type: HTTP http_probe { relative_url: "health" }`,
			wantErr: "invalid relative URL",
		},
		{
			name:     "http_bad_relative_url_redacted",
			probe:    `type: HTTP http_probe { relative_url: "health?token=secret" redact_url_query_in_logs: true }`,
			wantErr:  "invalid relative URL",
			notInErr: "secret",
		},
		{
			name:    "http_requests_too_long",
			probe:   `type: HTTP interval: "2s" timeout: "1s" http_probe { requests_per_probe: 10 requests_interval_msec: 500 }`,
			wantErr: "longer than the probe interval",
		},
		{
			// requests_per_probe * requests_interval_msec overflows int32.
			name:    "http_requests_too_long_overflow",
			probe:   `type: HTTP http_probe { requests_per_probe: 50000 requests_interval_msec: 50000 }`,
			wantErr: "longer than the probe interval",
		},
		{
			// requests_per_probe * requests_interval_msec in nanoseconds
			// overflows int64.
			name:    "http_requests_too_long_duration_overflow",
			probe:   `type: HTTP http_probe { requests_per_probe: 10000000 requests_interval_msec: 1000000 }`,
			wantErr: "longer than the probe interval",
		},
		{
			name:  "http_requests_fit_exactly",
			probe: `type: HTTP interval: "2s" timeout: "1s" http_probe { requests_per_probe: 10 requests_interval_msec: 100 }`,
		},
		{
			name:  "grpc_xds_not_validated",
			probe: `type: GRPC grpc_probe { uri_scheme: "xds:///" }`,
		},
		{
			name:    "http_bad_proxy_url",
			probe:   `type: HTTP http_probe { proxy_url: "ftp://proxy" }`,
			wantErr: "proxy_url",
		},
		{
			name:    "http_bad_response_metrics_options",
			probe:   `type: HTTP http_probe { response_metrics_options { line_accept_regex: "(" } }`,
			wantErr: "error initializing response metrics parser",
		},
		{
			name:  "dns_valid",
			probe: `type: DNS dns_probe { resolved_domain: "example.com" }`,
		},
		{
			name:    "dns_bad_query_type",
			probe:   `type: DNS dns_probe { query_type: NONE }`,
			wantErr: "invalid query type",
		},
		{
			name:    "dns_requests_too_long",
			probe:   `type: DNS interval: "2s" timeout: "1s" dns_probe { requests_per_probe: 10 requests_interval_msec: 500 }`,
			wantErr: "longer than the probe interval",
		},
		{
			name:  "ping_valid",
			probe: `type: PING`,
		},
		{
			name:    "ping_small_payload",
			probe:   `type: PING ping_probe { payload_size: 2 }`,
			wantErr: "cannot be smaller than",
		},
		{
			name:  "tcp_valid",
			probe: `type: TCP tcp_probe { tls_config {} }`,
		},
		{
			name:    "tcp_tls_without_handshake",
			probe:   `type: TCP tcp_probe { tls_handshake: false tls_config {} }`,
			wantErr: "tls_handshake is false",
		},
		{
			name:  "external_valid",
			probe: `type: EXTERNAL external_probe { command: "./probe.sh --arg 'a b'" output_as_metrics: true }`,
		},
		{
			name:    "external_no_command",
			probe:   `type: EXTERNAL external_probe { command: "" }`,
			wantErr: "command not specified",
		},
		{
			name:    "external_bad_command",
			probe:   `type: EXTERNAL external_probe { command: "echo 'x" }`,
			wantErr: "error parsing command line",
		},
		{
			name:    "external_bad_output_metrics_options",
			probe:   `type: EXTERNAL external_probe { command: "./probe.sh" output_as_metrics: true output_metrics_options { line_reject_regex: "(" } }`,
			wantErr: "error initializing payload metrics",
		},
		{
			name:  "sql_valid",
			probe: `type: SQL sql_probe { flavor: POSTGRES query_file: "/queries/q.sql" response_metrics_options {} }`,
		},
		{
			name:    "sql_no_flavor",
			probe:   `type: SQL sql_probe {}`,
			wantErr: "flavor is required",
		},
		{
			name:    "sql_query_and_query_file",
			probe:   `type: SQL sql_probe { flavor: POSTGRES query: "SELECT 1" query_file: "/queries/q.sql" }`,
			wantErr: "only one of query and query_file",
		},
		{
			name:    "sql_validators_without_query",
			probe:   `type: SQL sql_probe { flavor: POSTGRES } validator { name: "v" regex: "ok" }`,
			wantErr: "validators require a query",
		},
		{
			name:    "sql_response_metrics_without_query",
			probe:   `type: SQL sql_probe { flavor: POSTGRES response_metrics_options {} }`,
			wantErr: "response_metrics_options require a query",
		},
		{
			name:  "grpc_valid",
			probe: `type: GRPC grpc_probe { method: GENERIC request { body_file: "/requests/body.json" } }`,
		},
		{
			name:    "grpc_body_and_body_file",
			probe:   `type: GRPC grpc_probe { method: GENERIC request { body: "{}" body_file: "/requests/body.json" } }`,
			wantErr: "both body and body_file are set",
		},
		{
			name:  "external_grpc_valid",
			probe: `type: EXTERNAL_GRPC external_grpc_probe { server: "localhost:9000" probe_type: "redis" }`,
		},
		{
			name:    "external_grpc_no_server",
			probe:   `type: EXTERNAL_GRPC external_grpc_probe { probe_type: "redis" }`,
			wantErr: "server must be specified",
		},
		{
			name:    "external_grpc_no_probe_type",
			probe:   `type: EXTERNAL_GRPC external_grpc_probe { server: "localhost:9000" }`,
			wantErr: "probe_type must be specified",
		},
		{
			name:    "browser_requests_too_long",
			probe:   `type: BROWSER interval: "2s" timeout: "1s" browser_probe { requests_per_probe: 10 requests_interval_msec: 500 }`,
			wantErr: "longer than the probe interval",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &configpb.ProbeDef{}
			require.NoError(t, prototext.Unmarshal([]byte(`name: "test" targets { host_names: "localhost" } `+tt.probe), p))

			opts, err := options.ValidateProbeConfig(p)
			require.NoError(t, err)

			err = probes.ValidateConfig(p, opts)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				if tt.notInErr != "" {
					assert.NotContains(t, err.Error(), tt.notInErr)
				}
				return
			}
			assert.NoError(t, err)
		})
	}
}
