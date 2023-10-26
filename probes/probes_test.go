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
