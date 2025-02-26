// Copyright 2023-2025 The Cloudprober Authors.
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
	"fmt"
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/config/proto"
	"github.com/cloudprober/cloudprober/metrics"
	"github.com/cloudprober/cloudprober/probes"
	"github.com/cloudprober/cloudprober/probes/options"
	probes_configpb "github.com/cloudprober/cloudprober/probes/proto"
	testdatapb "github.com/cloudprober/cloudprober/probes/testdata"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestRandomDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		ceiling  time.Duration
	}{
		{
			duration: 0,
			ceiling:  10 * time.Second,
		},
		{
			duration: 5 * time.Second,
			ceiling:  10 * time.Second,
		},
		{
			duration: 30 * time.Second,
			ceiling:  10 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%v", tt), func(t *testing.T) {
			got := randomDuration(tt.duration, tt.ceiling)
			assert.LessOrEqual(t, got, tt.duration)
			assert.LessOrEqual(t, got, tt.ceiling)
		})
	}
}

func TestInterProbeWait(t *testing.T) {
	tests := []struct {
		interval  time.Duration
		numProbes int
		want      time.Duration
	}{
		{
			interval:  2 * time.Second,
			numProbes: 16,
			want:      125 * time.Millisecond,
		},
		{
			interval:  30 * time.Second,
			numProbes: 12,
			want:      2 * time.Second,
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s:%d", tt.interval, tt.numProbes), func(t *testing.T) {
			assert.Equal(t, tt.want, interProbeWait(tt.interval, tt.numProbes))
		})
	}
}

// testProbe implements the probes.Probe interface, while providing
// facilities to examine the probe status for the purpose of testing.
// Since cloudprober has to be aware of the probe type, we add testProbe to
// cloudprober as an EXTENSION probe type (done through the init() function
// below).
type testProbe struct {
	intialized      bool
	startTime       time.Time
	runningStatusCh chan bool
}

// We use an EXTENSION probe for testing. Following has the same effect as:
// This has the same effect as using the following in your config:
//
//	probe {
//	   name: "<name>"
//	   targets {
//	    dummy_targets{}
//	   }
//	   [cloudprober.probes.testdata.fancy_probe] {
//	     name: "fancy"
//	   }
//	}
func testProbeDef(name string) *probes_configpb.ProbeDef {
	probeDef := &probes_configpb.ProbeDef{
		Name: proto.String(name),
		Type: probes_configpb.ProbeDef_EXTENSION.Enum(),
		Targets: &targetspb.TargetsDef{
			Type: &targetspb.TargetsDef_DummyTargets{},
		},
	}
	proto.SetExtension(probeDef, testdatapb.E_FancyProbe, &testdatapb.FancyProbe{Name: proto.String("fancy-" + name)})
	return probeDef
}

func (p *testProbe) Init(name string, opts *options.Options) error {
	p.intialized = true
	p.runningStatusCh = make(chan bool)
	return nil
}

func (p *testProbe) Start(ctx context.Context, dataChan chan *metrics.EventMetrics) {
	p.startTime = time.Now()

	p.runningStatusCh <- true

	// If context is done (used to stop a running probe before removing it),
	// change probe state to not-running.
	<-ctx.Done()
	p.runningStatusCh <- false
	close(p.runningStatusCh)
}

func init() {
	// Register extension probe.
	probes.RegisterProbeType(200, func() probes.Probe {
		return &testProbe{}
	})
}

func TestStartProbesWithJitter(t *testing.T) {
	pr, cancel := testProber(t, &configpb.ProberConfig{
		Probe: []*probes_configpb.ProbeDef{
			testProbeDef("test-probe-1"),
			testProbeDef("test-probe-2"),
		},
	})
	defer cancel()

	var startTimes []time.Time
	for _, p := range pr.Probes {
		tp := p.Probe.(*testProbe)
		<-tp.runningStatusCh
		startTimes = append(startTimes, tp.startTime)
	}
	assert.Equal(t, 2, len(startTimes))
	delay := startTimes[1].Sub(startTimes[0])
	if delay < 0 {
		delay = -delay
	}
	assert.GreaterOrEqual(t, delay, time.Second)
}
