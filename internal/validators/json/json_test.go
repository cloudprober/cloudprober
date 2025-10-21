// Copyright 2022 The Cloudprober Authors.
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

package json

import (
	"testing"

	configpb "github.com/cloudprober/cloudprober/internal/validators/json/proto"
	"github.com/stretchr/testify/assert"
)

var testInput = `
{
  "next": null,
  "previous": null,
  "results": [
    {
      "active": true,
      "id": "3d961844",
      "state": "queued"
    },
    {
      "active": true,
      "id": "6f2afc32",
      "state": "sell_only"
    },
    {
      "active": true,
      "id": "6f2afc33",
      "state": "sell_only"
    }
  ]
}
`

func TestValidate(t *testing.T) {
	var tests = []struct {
		desc  string
		jqf   string
		input string

		// Expected results
		parseErr bool
		retFalse bool
		retError bool
	}{
		{
			desc: "no_jq_filter",
		},
		{
			desc:     "no_jq_filter_bad_input",
			input:    "{'a': 'b',}",
			retFalse: true,
			retError: true,
		},
		{
			desc: "jq_filter_state_sell_only",
			jqf:  "[.results[] | select(.state == \"sell_only\") | .id] | length == 2",
		},
		{
			desc: "jq_filter_expected_vals",
			jqf:  "[.results[] | select(.state == \"sell_only\") | .id] | [\"6f2afc32\",\"6f2afc33\"] - . | length == 0",
		},
		{
			desc: "jq_filter_state_queued",
			jqf:  "[.results[] | select(.state == \"queued\") | .id] | length == 1",
		},
		{
			desc:     "jq_filter_not_returning_bool",
			jqf:      "[.results[] | select(.state == \"sell_only\") | .id]",
			retFalse: true,
			retError: true,
		},
		{
			desc:     "jq_filter_not_returning_true",
			jqf:      "[.results[] | select(.state == \"sell_only\") | .id] | length == 1",
			retFalse: true,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			v := Validator{}
			err := v.Init(&configpb.Validator{
				JqFilter: test.jqf,
			})

			if test.parseErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			if err != nil {
				return
			}

			if test.input == "" {
				test.input = testInput
			}

			ret, err := v.Validate([]byte(test.input), nil)
			if test.retError {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}
			assert.Equal(t, !test.retFalse, ret)
		})
	}
}
