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

package probestatus

import (
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestDashboardDurations(t *testing.T) {
	testData := map[time.Duration][]time.Duration{
		24 * time.Hour:  baseDurations[:6],
		72 * time.Hour:  baseDurations[:7],
		168 * time.Hour: baseDurations[:8],
		360 * time.Hour: append(append([]time.Duration{}, baseDurations[:8]...), 360*time.Hour),
		720 * time.Hour: baseDurations,
	}

	for maxD, wantDurations := range testData {
		t.Run(maxD.String(), func(t *testing.T) {
			durations, _ := dashboardDurations(maxD)
			if !reflect.DeepEqual(durations, wantDurations) {
				t.Errorf("Got durations=%v, wanted=%v", durations, wantDurations)
			}
		})
	}
}

func TestShortDur(t *testing.T) {
	testData := map[time.Duration]string{
		30 * time.Minute: "30m",
		24 * time.Hour:   "24h",
		72 * time.Hour:   "3d",
		96 * time.Hour:   "4d",
		168 * time.Hour:  "7d",
		720 * time.Hour:  "30d",
	}

	for td, want := range testData {
		t.Run(td.String(), func(t *testing.T) {
			got := shortDur([]time.Duration{td})[0]
			if got != want {
				t.Errorf("Got string=%s, wanted=%s", got, want)
			}
		})
	}
}

func TestTrimDurations(t *testing.T) {
	dashDurations := []time.Duration{time.Minute, 5 * time.Minute, time.Hour, 6 * time.Hour}

	tests := map[string][]time.Duration{
		"24h": dashDurations,
		"5h":  dashDurations,
		"45m": dashDurations[:3],
		"3m":  dashDurations[:2],
		"10s": dashDurations[:1],
	}

	for maxIntervalStr, wantDurations := range tests {
		t.Run(fmt.Sprintf("maxInterval: %s", maxIntervalStr), func(t *testing.T) {
			maxInterval, _ := time.ParseDuration(maxIntervalStr)
			durations := trimDurations(dashDurations, maxInterval)
			assert.Equal(t, wantDurations, durations)
		})
	}
}
