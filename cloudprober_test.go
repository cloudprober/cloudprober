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
	"errors"
	"fmt"
	"net"
	"os"
	"testing"
	"time"

	"github.com/cloudprober/cloudprober/config"
	configpb "github.com/cloudprober/cloudprober/config/proto"
	serverspb "github.com/cloudprober/cloudprober/internal/servers/proto"
	udpserverpb "github.com/cloudprober/cloudprober/internal/servers/udp/proto"
	surfacerspb "github.com/cloudprober/cloudprober/internal/surfacers/proto"
	"github.com/cloudprober/cloudprober/internal/tracing"
	"github.com/cloudprober/cloudprober/metrics"
	probepb "github.com/cloudprober/cloudprober/probes/proto"
	udpprobepb "github.com/cloudprober/cloudprober/probes/udp/proto"
	"github.com/cloudprober/cloudprober/state"
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
	// Check the context before the select below: with room left in the channel
	// and a context that's already done, select picks between them at random.
	if ctx.Err() != nil {
		return
	}
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

// Cloudprober can be initialized only once, so tests that initialize it have
// to put the globals back by hand.
func resetCloudProber(t *testing.T) {
	t.Helper()
	Shutdown()
	cloudProber.Lock()
	defer cloudProber.Unlock()
	cloudProber.instance = instance{}
	cloudProber.done = false
	state.SetDefaultGRPCServer(nil)
	state.SetDefaultHTTPServeMux(nil)
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

func TestShutdown(t *testing.T) {
	t.Run("no tracing configured", func(t *testing.T) {
		resetCloudProber(t)
		setTracingShutdown(nil)
		Shutdown() // Should be a no-op.
	})

	t.Run("flushes spans once", func(t *testing.T) {
		resetCloudProber(t)

		var calls int
		var deadline time.Time
		setTracingShutdown(func(ctx context.Context) error {
			calls++
			deadline, _ = ctx.Deadline()
			return nil
		})

		Shutdown()
		assert.Equal(t, 1, calls, "tracing shutdown calls")
		assert.WithinDuration(t, time.Now().Add(tracingShutdownTimeout), deadline, time.Second, "tracing shutdown deadline")

		// Shutdown is safe to call again, but doesn't shut tracing down twice.
		Shutdown()
		assert.Equal(t, 1, calls, "tracing shutdown calls after second Shutdown")
	})

	t.Run("logs shutdown error", func(t *testing.T) {
		resetCloudProber(t)
		setTracingShutdown(func(ctx context.Context) error {
			return errors.New("collector unreachable")
		})
		Shutdown() // Error is logged, not returned.

		cloudProber.RLock()
		defer cloudProber.RUnlock()
		assert.Nil(t, cloudProber.tracingShutdown, "tracingShutdown after error")
	})
}

func setTracingShutdown(f tracing.ShutdownFunc) {
	cloudProber.Lock()
	defer cloudProber.Unlock()
	cloudProber.tracingShutdown = f
}

// Verify that Shutdown releases what Init acquired, even if Start was never
// called -- the RunOnce path.
func TestShutdownWithoutStart(t *testing.T) {
	resetCloudProber(t)

	port := freePortsT(t, 1)[0]

	f, err := os.CreateTemp("", "cloudprober_test")
	if err != nil {
		t.Fatalf("os.CreateTemp(): %v", err)
	}
	defer os.Remove(f.Name())
	cfg := &configpb.ProberConfig{Port: proto.Int32(port)}
	os.WriteFile(f.Name(), []byte(prototext.Format(cfg)), 0644)

	if err := InitWithConfigSource(config.ConfigSourceWithFile(f.Name())); err != nil {
		t.Fatalf("InitWithConfigSource(): %v", err)
	}

	// Init opens the default HTTP server's listener, whether or not we Start.
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err == nil {
		ln.Close()
		t.Fatalf("port %d is free after Init, expected it to be in use", port)
	}

	Shutdown()

	assert.Nil(t, GetProber(), "prober after Shutdown")
	assert.EqualError(t, RunOnce(context.Background(), "", "text", "  "), "cloudprober is not initialized", "RunOnce() after Shutdown")

	ln, err = net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		t.Fatalf("port %d is still in use after Shutdown: %v", port, err)
	}
	ln.Close()
}

// Verify that Shutdown stops what Start started, even when the context passed
// to Start is still live. The prober's probes and its surfacer writes run on
// the start context (the servers' sockets, in contrast, are tied to the init
// context), so a surfacer that goes quiet is the signal that Shutdown reached
// it.
func TestShutdownStopsStartedProber(t *testing.T) {
	resetCloudProber(t)

	ports := freePortsT(t, 2)

	f, err := os.CreateTemp("", "cloudprober_test")
	if err != nil {
		t.Fatalf("os.CreateTemp(): %v", err)
	}
	defer os.Remove(f.Name())
	cfg := &configpb.ProberConfig{
		Port: proto.Int32(ports[0]),
		Server: []*serverspb.ServerDef{
			{
				Type: serverspb.ServerDef_UDP.Enum(),
				Server: &serverspb.ServerDef_UdpServer{
					UdpServer: &udpserverpb.ServerConf{
						Port: proto.Int32(ports[1]),
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
				IntervalMsec:            proto.Int32(10),
				StatsExportIntervalMsec: proto.Int32(int32(500 * time.Millisecond / time.Millisecond)),
				Targets: &targetspb.TargetsDef{
					Type: &targetspb.TargetsDef_HostNames{HostNames: "localhost"},
				},
				Probe: &probepb.ProbeDef_UdpProbe{
					UdpProbe: &udpprobepb.ProbeConf{
						Port:        proto.Int32(ports[1]),
						PayloadSize: proto.Int32(10),
					},
				},
			},
		},
		Surfacer: []*surfacerspb.SurfacerDef{
			{
				Name: proto.String("custom"),
				Type: surfacerspb.Type_USER_DEFINED.Enum(),
			},
		},
	}
	if err := os.WriteFile(f.Name(), []byte(prototext.Format(cfg)), 0644); err != nil {
		t.Fatalf("os.WriteFile(): %v", err)
	}

	fs := &FakeSurfacer{c: make(chan *metrics.EventMetrics, 10)}
	surfacers.Register("custom", fs)

	if err := InitWithConfigSource(config.ConfigSourceWithFile(f.Name())); err != nil {
		t.Fatalf("InitWithConfigSource(): %v", err)
	}

	// Note the background context: nothing but Shutdown can stop this prober.
	Start(context.Background())
	srvAddr := cloudProber.defaultServerLn.Addr().String()

	select {
	case <-time.After(30 * time.Second):
		t.Fatal("surfacer timed out before getting results")
	case <-fs.c:
	}

	Shutdown()

	// Drain what the surfacer had already queued before the shutdown.
	time.Sleep(200 * time.Millisecond)
	for len(fs.c) > 0 {
		<-fs.c
	}

	// FakeSurfacer.Write drops metrics once its context is done, so anything
	// arriving now means the prober is still running.
	select {
	case em := <-fs.c:
		t.Errorf("got metrics after Shutdown: %s", em.String())
	case <-time.After(2 * time.Second):
	}

	// And the default server gives its port back.
	ln, err := net.Listen("tcp", srvAddr)
	if err != nil {
		t.Fatalf("default server's port is still in use after Shutdown: %v", err)
	}
	ln.Close()
}

// Cloudprober can't be brought back up once it's been shut down.
func TestInitAfterShutdownPanics(t *testing.T) {
	resetCloudProber(t)

	f, err := os.CreateTemp("", "cloudprober_test")
	if err != nil {
		t.Fatalf("os.CreateTemp(): %v", err)
	}
	defer os.Remove(f.Name())
	cfg := &configpb.ProberConfig{Port: proto.Int32(freePortsT(t, 1)[0])}
	if err := os.WriteFile(f.Name(), []byte(prototext.Format(cfg)), 0644); err != nil {
		t.Fatalf("os.WriteFile(): %v", err)
	}

	if err := InitWithConfigSource(config.ConfigSourceWithFile(f.Name())); err != nil {
		t.Fatalf("InitWithConfigSource(): %v", err)
	}
	assert.Panics(t, func() { InitWithConfigSource(config.ConfigSourceWithFile(f.Name())) }, "Init() while initialized")

	Shutdown()
	assert.Panics(t, func() { InitWithConfigSource(config.ConfigSourceWithFile(f.Name())) }, "Init() after Shutdown()")
}
