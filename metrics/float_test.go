// Copyright 2023 The Cloudprober Authors.
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
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFloat(t *testing.T) {
	tests := []struct {
		f          float64
		strFunc    func(float64) string
		wantInt64  int64
		wantFloat  float64
		wantString string
	}{
		{
			f:          1.234,
			wantInt64:  1,
			wantFloat:  1.234,
			wantString: "1.234",
		},
		{
			f:          1.2345,
			wantInt64:  1,
			wantFloat:  1.2345,
			wantString: "1.234", // Floats get truncated to 3 decimal places.
		},
		{
			f:          1.2345,
			wantInt64:  1,
			wantFloat:  1.2345,
			strFunc:    func(f float64) string { return fmt.Sprintf("%.4f", f) },
			wantString: "1.2345", // Floats get truncated to 3 decimal places.
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("float:%v", tt.f), func(t *testing.T) {
			f := NewFloat(tt.f)
			if tt.strFunc != nil {
				f.Str = tt.strFunc
			}
			assert.Equal(t, tt.wantInt64, f.Int64())
			assert.Equal(t, tt.wantFloat, f.Float64())
			assert.Equal(t, tt.wantString, f.String())
		})
	}
}
