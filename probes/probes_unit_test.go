package probes

import (
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/proto"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"testing"
)

// This test is to make sure that we don't panic on empty config.
// https://github.com/cloudprober/cloudprober/issues/293
func TestProbesAllowEmptyConfig(t *testing.T) {
	for probeName, probeValue := range configpb.ProbeDef_Type_value {
		t.Run(probeName, func(t *testing.T) {
			probeDef := &configpb.ProbeDef{
				Name: &probeName,
				Type: configpb.ProbeDef_Type(probeValue).Enum(),
				Targets: &targetspb.TargetsDef{
					Type: &targetspb.TargetsDef_DummyTargets{
						DummyTargets: &targetspb.DummyTargets{},
					},
				},
			}

			opts, err := options.BuildProbeOptions(probeDef, nil, nil, nil)
			if err != nil {
				t.Errorf("error building probe options: %v", err)
			}

			// ensure that we don't panic on empty config
			_, _, _ = initProbe(probeDef, opts)
		})
	}
}
