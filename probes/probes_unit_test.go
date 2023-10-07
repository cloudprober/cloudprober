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

// This file is for internal unit tests for probes package. It was
// created to test the fix for the following issue:
// https://github.com/cloudprober/cloudprober/issues/293

package probes

import (
	"github.com/cloudprober/cloudprober/probes/options"
	configpb "github.com/cloudprober/cloudprober/probes/proto"
	targetspb "github.com/cloudprober/cloudprober/targets/proto"
	"testing"
)

// This test is to make sure that we don't panic on empty config.
// https://github.com/cloudprober/cloudprober/issues/293
// This test utilizes the `probes.initProbe` function, which is not exported by
// the probes package.
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
