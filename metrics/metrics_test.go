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

package metrics

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseValueFromString(t *testing.T) {
	var tests = []struct {
		val      string
		wantType Value
		wantErr  bool
	}{
		{
			val:     "234g",
			wantErr: true,
		},
		{
			val:      "234",
			wantType: NewFloat(1),
		},
		{
			val:      "-234",
			wantType: NewFloat(1),
		},
		{
			val:      ".234",
			wantType: NewFloat(1),
		},
		{
			val:      "-.234",
			wantType: NewFloat(1),
		},
		{
			val:     ".-234",
			wantErr: true,
		},
		{
			val:      "\"234\"",
			wantType: NewString(""),
		},
		{
			val:      "map:code 200:10 404:1",
			wantType: NewMapFloat("m"),
		},
		{
			val:      "dist:sum:899|count:221|lb:-Inf,0.5,2,7.5|bc:34,54,121,12",
			wantType: NewDistribution([]float64{1}),
		},
	}

	for _, test := range tests {
		t.Run("value="+test.val, func(t *testing.T) {
			v, err := ParseValueFromString(test.val)

			if err != nil {
				if !test.wantErr {
					t.Errorf("Got unexpected error: %v", err)
				}
				return
			}
			if test.wantErr {
				t.Errorf("Didn't get the expected error, got parsed value: %v", v)
				return
			}
			assert.IsType(t, test.wantType, v)
		})
	}
}
