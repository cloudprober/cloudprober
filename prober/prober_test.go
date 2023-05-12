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

package prober

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
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
