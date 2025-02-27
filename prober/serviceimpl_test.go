// Copyright 2019-2025 The Cloudprober Authors.
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

package prober

import (
	"context"
	"net/http"
	"os"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/stretchr/testify/assert"

	pb "github.com/cloudprober/cloudprober/prober/proto"
	probes_configpb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/cloudprober/cloudprober/state"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/proto"
)

func testProber(t *testing.T, cfg *configpb.ProberConfig) (*Prober, context.CancelFunc) {
	t.Helper()

	// We need this to initialize default surfacers.
	httpServerMux := http.NewServeMux()
	state.SetDefaultHTTPServeMux(httpServerMux)
	defer state.SetDefaultHTTPServeMux(nil)

	// We need this to acitvate the gRPC functionality.
	grpcServer := grpc.NewServer()
	state.SetDefaultGRPCServer(grpcServer)
	defer state.SetDefaultGRPCServer(nil)

	ctx, cancel := context.WithCancel(context.Background())
	pr, err := Init(ctx, cfg, nil)
	if err != nil {
		t.Fatalf("error while initializing prober: %v", err)
	}

	pr.Start(ctx)
	return pr, cancel
}

// verifyProbeRunningStatus is a helper function to verify probe's running
// status.
func verifyProbeRunningStatus(t *testing.T, p *testProbe, expectedRunning bool) {
	t.Helper()

	// We use large timeout because we should never really timeout in ideal
	// conditions.
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	var running, timedout bool

	select {
	case running = <-p.runningStatusCh:
	case <-ctx.Done():
		timedout = true
	}

	if timedout {
		t.Errorf("timed out while waiting for test probe's running status")
		return
	}

	if running != expectedRunning {
		t.Errorf("Running status: expected=%v, got=%v", expectedRunning, running)
	}
}

func TestAddProbe(t *testing.T) {
	pr, cancel := testProber(t, &configpb.ProberConfig{})
	defer cancel()

	// Test AddProbe()
	// Empty probe config should return an error
	_, err := pr.AddProbe(context.Background(), &pb.AddProbeRequest{})
	if err == nil {
		t.Error("empty probe config didn't result in error")
	}

	// Add a valid test probe
	testProbeName := "test-probe"
	_, err = pr.AddProbe(context.Background(), &pb.AddProbeRequest{ProbeConfig: testProbeDef(testProbeName)})
	if err != nil {
		t.Errorf("error while adding the probe: %v", err)
	}
	if pr.Probes[testProbeName] == nil {
		t.Errorf("test-probe not added to the probes database")
	}

	p := pr.Probes[testProbeName].Probe.(*testProbe)
	if !p.intialized {
		t.Errorf("test probe not initialized. p.initialized=%v", p.intialized)
	}
	verifyProbeRunningStatus(t, p, true)

	// Add test probe again, should result in error
	_, err = pr.AddProbe(context.Background(), &pb.AddProbeRequest{ProbeConfig: testProbeDef(testProbeName)})
	if err == nil {
		t.Error("adding same probe didn't result in error")
	}
}

func TestListProbes(t *testing.T) {
	pr, cancel := testProber(t, &configpb.ProberConfig{})
	defer cancel()

	// Add couple of probes for testing
	testProbes := []string{"test-probe-1", "test-probe-2"}
	for _, name := range testProbes {
		pr.AddProbe(context.Background(), &pb.AddProbeRequest{ProbeConfig: testProbeDef(name)})
	}

	// Test ListProbes()
	resp, err := pr.ListProbes(context.Background(), &pb.ListProbesRequest{})
	if err != nil {
		t.Errorf("error while list probes: %v", err)
	}
	respProbes := resp.GetProbe()
	if len(respProbes) != len(testProbes) {
		t.Errorf("didn't get correct number of probe in ListProbes response. Got %d probes, expected %d probes", len(respProbes), len(testProbes))
	}
	var respProbeNames []string
	for _, p := range respProbes {
		respProbeNames = append(respProbeNames, p.GetName())
		t.Logf("Probe Config: %+v", p.GetConfig())
	}
	sort.Strings(respProbeNames)
	if !reflect.DeepEqual(respProbeNames, testProbes) {
		t.Errorf("Probes in ListProbes() response: %v, expected: %s", respProbeNames, testProbes)
	}
}

func TestRemoveProbes(t *testing.T) {
	pr, cancel := testProber(t, &configpb.ProberConfig{})
	defer cancel()

	testProbeName := "test-probe-for-remove"

	// Remove a non-existent probe, should result in error.
	_, err := pr.RemoveProbe(context.Background(), &pb.RemoveProbeRequest{ProbeName: &testProbeName})
	if err == nil {
		t.Error("removing non-existent probe didn't result in error")
	}

	// Add a probe for testing
	_, err = pr.AddProbe(context.Background(), &pb.AddProbeRequest{ProbeConfig: testProbeDef(testProbeName)})
	if err != nil {
		t.Errorf("error while adding test probe: %v", err)
	}
	p := pr.Probes[testProbeName].Probe.(*testProbe)
	verifyProbeRunningStatus(t, p, true)

	_, err = pr.RemoveProbe(context.Background(), &pb.RemoveProbeRequest{ProbeName: &testProbeName})
	if err != nil {
		t.Errorf("error while removing probe: %v", err)
	}

	if pr.Probes[testProbeName] != nil {
		t.Errorf("test probe still in the probes database: %v", pr.Probes[testProbeName])
	}

	// Verify that probe is not running anymore
	verifyProbeRunningStatus(t, p, false)
}

func TestSaveProbesConfig(t *testing.T) {
	tmpFile := func() *os.File {
		f, err := os.CreateTemp(t.TempDir(), "cloudprober_save.cfg")
		if err != nil {
			t.Fatalf("error while creating temp file: %v", err)
		}
		f.Close()
		return f
	}

	f := tmpFile()
	*probesConfigSavePath = f.Name()
	defer func() {
		*probesConfigSavePath = ""
	}()

	targetsDef := &targetspb.TargetsDef{
		Type: &targetspb.TargetsDef_HostNames{
			HostNames: "manugarg.com",
		},
	}

	pr, cancel := testProber(t, &configpb.ProberConfig{
		Probe: []*probes_configpb.ProbeDef{
			{
				Name:    proto.String("test-probe-1"),
				Type:    probes_configpb.ProbeDef_PING.Enum(),
				Targets: targetsDef,
			},
		},
	})
	defer cancel()

	if _, err := pr.AddProbe(context.Background(), &pb.AddProbeRequest{
		ProbeConfig: &probes_configpb.ProbeDef{
			Name:    proto.String("test-probe-2"),
			Type:    probes_configpb.ProbeDef_PING.Enum(),
			Targets: targetsDef,
		},
	}); err != nil {
		t.Errorf("error while adding test probe: %v", err)
	}

	wantConfigProbe1 := "probe: {\n  name: \"test-probe-1\"\n  type: PING\n  targets: {\n    host_names: \"manugarg.com\"\n  }\n}\n"
	wantConfigProbe2 := strings.ReplaceAll(wantConfigProbe1, "test-probe-1", "test-probe-2")

	compareConfig := func(fileName string, wantConfig string) {
		b, err := os.ReadFile(fileName)
		if err != nil {
			t.Errorf("error while reading saved config file: %v", err)
		}
		gotConfig := strings.ReplaceAll(string(b), ":  ", ": ")
		assert.Equal(t, wantConfig, gotConfig)
	}

	compareConfig(*probesConfigSavePath, wantConfigProbe1+wantConfigProbe2)

	// Remove a probe and check again
	if _, err := pr.RemoveProbe(context.Background(), &pb.RemoveProbeRequest{ProbeName: proto.String("test-probe-2")}); err != nil {
		t.Errorf("error while removing probe: %v", err)
	}
	compareConfig(*probesConfigSavePath, wantConfigProbe1)

	// Test SaveProbeConfig
	f2 := tmpFile()
	pr.SaveProbesConfig(context.Background(), &pb.SaveProbesConfigRequest{FilePath: proto.String(f2.Name())})
	compareConfig(f2.Name(), wantConfigProbe1)
}
