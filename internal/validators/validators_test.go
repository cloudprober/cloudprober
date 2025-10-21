// Copyright 2019 The Cloudprober Authors.
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

package validators

import (
	"reflect"
	"testing"

	configpb "github.com/cloudprober/cloudprober/internal/validators/proto"
	"github.com/cloudprober/cloudprober/logger"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/encoding/prototext"
)

var testValidators = []*Validator{
	{
		Name:     "test-v1",
		Validate: func(input *Input, l *logger.Logger) (bool, error) { return true, nil },
	},
	{
		Name:     "test-v2",
		Validate: func(input *Input, l *logger.Logger) (bool, error) { return false, nil },
	},
}

func TestRunValidators(t *testing.T) {
	// Try with no metrics map
	assert.Equal(t, []string{"test-v2"}, RunValidators(testValidators, &Input{}, nil, nil))

	// Try with metrics map
	vfMap := ValidationFailureMap(testValidators)
	assert.Equal(t, []string{"test-v2"}, RunValidators(testValidators, &Input{}, vfMap, nil))

	if vfMap.GetKey("test-v1") != 0 {
		t.Error("Got unexpected test-v1 validation failure.")
	}

	if vfMap.GetKey("test-v2") != 1 {
		t.Errorf("Didn't get expected test-v2 validation failure.")
	}
}

func TestValidatorFailureMap(t *testing.T) {
	vfMap := ValidationFailureMap(testValidators)

	expectedKeys := []string{"test-v1", "test-v3"}
	if reflect.DeepEqual(vfMap.Keys(), expectedKeys) {
		t.Errorf("Didn't get expected keys in the mao. Got: %s, Expected: %v", vfMap.Keys(), expectedKeys)
	}
}

func TestInit(t *testing.T) {
	tests := []struct {
		name           string
		validatorConfs []string
		wantNames      []string
		wantErr        string
	}{
		{
			name: "success",
			validatorConfs: []string{
				`
					name: "http_status_200s"
				    http_validator {
						success_status_codes: "200-299"
					}
				`,
				`
					name: "found_string"
					regex: ".*found.*"
				`,
				`
					name: "valid_json"
					json_validator {
					}
				`,
				`
					name: "integrity"
					integrity_validator {
						pattern_num_bytes: 8
					}
				`,
			},
			wantNames: []string{"http_status_200s", "found_string", "valid_json", "integrity"},
		},
		{
			name: "missing name",
			validatorConfs: []string{
				`http_validator {
					success_status_codes: "200-299"
				}`,
			},
			wantErr: "validator name",
		},
		{
			name: "invalid validator config",
			validatorConfs: []string{
				`
				name: "http_status_200s"
				http_validator {
				}`,
			},
			wantErr: "HTTP",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var validators []*configpb.Validator
			for _, validatorConf := range tt.validatorConfs {
				v := &configpb.Validator{}
				prototext.Unmarshal([]byte(validatorConf), v)
				validators = append(validators, v)
			}

			got, err := Init(validators)
			if err != nil {
				if tt.wantErr == "" {
					t.Errorf("Init() unexpected error: %v", err)
				}
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}

			if err == nil && tt.wantErr != "" {
				t.Errorf("Init() expected error: %s, got nil", tt.wantErr)
			}

			var gotNames []string
			for _, v := range got {
				gotNames = append(gotNames, v.Name)
			}
			assert.Equal(t, tt.wantNames, gotNames)
		})
	}
}
