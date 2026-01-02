// Copyright 2019-2020 The Cloudprober Authors.
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

package cloudprober

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/config"
	configpb "github.com/cloudprober/cloudprober/config/proto"
	serverspb "github.com/cloudprober/cloudprober/internal/servers/proto"
	udpserverpb "github.com/cloudprober/cloudprober/internal/servers/udp/proto"
	surfacerspb "github.com/cloudprober/cloudprober/internal/surfacers/proto"
	"github.com/cloudprober/cloudprober/metrics"
	probepb "github.com/cloudprober/cloudprober/probes/proto"
	udpprobepb "github.com/cloudprober/cloudprober/probes/udp/proto"
	"github.com/cloudprober/cloudprober/surfacers"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
	"google.golang.org/protobuf/proto"
)

func TestGetDefaultServerPort(t *testing.T) {
	tests := []struct {
		desc       string
		configPort int32
		envVar     string
		wantPort   int
		wantErr    bool
	}{
		{
			desc:       "use port from config",
			configPort: 9316,
			envVar:     "3141",
			wantPort:   9316,
		},
		{
			desc:       "use default port",
			configPort: 0,
			envVar:     "",
			wantPort:   DefaultServerPort,
		},
		{
			desc:       "use port from env",
			configPort: 0,
			envVar:     "3141",
			wantPort:   3141,
		},
		{
			desc:       "ignore kubernetes port",
			configPort: 0,
			envVar:     "tcp://100.101.102.103:3141",
			wantPort:   9313,
		},
		{
			desc:       "error due to bad env var",
			configPort: 0,
			envVar:     "a3141",
			wantErr:    true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			os.Setenv(ServerPortEnvVar, test.envVar)
			port, err := getDefaultServerPort(&configpb.ProberConfig{
				Port: proto.Int32(test.configPort),
			}, nil)

			if err != nil {
				if !test.wantErr {
					t.Errorf("Got unexpected error: %v", err)
				} else {
					return
				}
			}

			if port != test.wantPort {
				t.Errorf("got port: %d, want port: %d", port, test.wantPort)
			}
		})
	}

}

type FakeSurfacer struct {
	c chan *metrics.EventMetrics
}

func (f *FakeSurfacer) Write(ctx context.Context, em *metrics.EventMetrics) {
	if em.Label("ptype") != "udp" {
		return
	}
	select {
	case f.c <- em:
	case <-ctx.Done():
	}
}

func freePortsT(t *testing.T, n int) []int32 {
	ports := make([]int32, 0, n)
	for i := 0; i < n; i++ {
		l, err := net.Listen("tcp", ":0")
		if err != nil {
			t.Fatalf("net.Listen(%q, %q): %v", "tcp", ":0", err)
		}
		defer l.Close()
		ports = append(ports, int32(l.Addr().(*net.TCPAddr).Port))
	}
	return ports
}

func TestRestart(t *testing.T) {
	type TestCase struct {
		Name string
	}
	testCases := []TestCase{
		{
			Name: "FirstInitAndStart",
		},
		{
			Name: "SecondInitAndStart",
		},
	}

	// We reuse ports to verify that we're cleaning up properly after each run.
	ports := freePortsT(t, 3)

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer func() {
				cancel()
				// Wait required for the cloudprober instance to fully shut down.
				time.Sleep(time.Second)
			}()
			cfg := &configpb.ProberConfig{
				Port:     proto.Int32(ports[0]),
				GrpcPort: proto.Int32(ports[1]),
				Server: []*serverspb.ServerDef{
					{
						Type: serverspb.ServerDef_UDP.Enum(),
						Server: &serverspb.ServerDef_UdpServer{
							UdpServer: &udpserverpb.ServerConf{
								Port: proto.Int32(ports[2]),
								Type: udpserverpb.ServerConf_ECHO.Enum(),
							},
						},
					},
				},
				Probe: []*probepb.ProbeDef{
					{
						Name:                    proto.String("udp echo"),
						Type:                    probepb.ProbeDef_UDP.Enum(),
						TimeoutMsec:             proto.Int32(10),
						IntervalMsec:            proto.Int32(10), // 100 probe per second
						StatsExportIntervalMsec: proto.Int32(int32(1 * time.Second / time.Millisecond)),
						Targets: &targetspb.TargetsDef{
							Type: &targetspb.TargetsDef_HostNames{
								HostNames: "localhost",
							},
						},
						Probe: &probepb.ProbeDef_UdpProbe{
							UdpProbe: &udpprobepb.ProbeConf{
								Port:        proto.Int32(ports[2]),
								PayloadSize: proto.Int32(10),
							},
						},
					},
				},
			}
			surfacerName := "custom"
			cfg.Surfacer = []*surfacerspb.SurfacerDef{
				{
					Name: proto.String(surfacerName),
					Type: surfacerspb.Type_USER_DEFINED.Enum(),
				},
			}
			s := &FakeSurfacer{c: make(chan *metrics.EventMetrics, 10)}
			surfacers.Register(surfacerName, s)

			tmpfile, err := os.CreateTemp("", "cloudprober_test")
			if err != nil {
				t.Fatalf("os.CreateTemp(): %v", err)
			}
			defer os.Remove(tmpfile.Name())
			os.WriteFile(tmpfile.Name(), []byte(prototext.Format(cfg)), 0644)

			err = InitWithConfigSource(config.ConfigSourceWithFile(tmpfile.Name()))
			if err != nil {
				t.Fatalf("Err: %v, Config: %s", err, prototext.Format(cfg))
			}
			Start(ctx)

			// Wait for results from the surfacer, or 30s,
			// whichever comes first. Since the export rate is 1s,
			// we should expect results well before 30s has passed.
			select {
			case <-time.After(time.Second * 30):
				t.Fatal("surfacer timed out before getting results")
			case <-s.c:
			}
		})
	}
}

func TestCloudproberConfig(t *testing.T) {
	rawCfg := `probe { type: PING, name: "test_probe", targets { host_names: "localhost" }}`
	f, err := os.CreateTemp("", "cloudprober_test")
	if err != nil {
		t.Fatalf("os.CreateTemp(): %v", err)
	}
	defer os.Remove(f.Name())
	os.WriteFile(f.Name(), []byte(rawCfg), 0644)

	tests := []struct {
		name             string
		fileName         string
		wantProbename    string
		wantRawConfig    string
		wantParsedConfig string
	}{
		{
			name:             "config from file",
			fileName:         f.Name(),
			wantProbename:    "test_probe",
			wantRawConfig:    rawCfg,
			wantParsedConfig: rawCfg,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			configSrc := config.ConfigSourceWithFile(tt.fileName)

			cloudProber.Lock()
			cloudProber.configSource = configSrc
			cloudProber.config, _ = configSrc.GetConfig()
			cloudProber.Unlock()

			assert.Equal(t, tt.wantProbename, GetConfig().GetProbe()[0].GetName(), "GetConfig()")
			assert.Equal(t, tt.wantRawConfig, GetRawConfig(), "GetRawConfig()")
			assert.Equal(t, tt.wantParsedConfig, GetParsedConfig(), "GetParsedConfig()")
		})
	}
}
