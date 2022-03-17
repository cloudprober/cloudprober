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
	"strings"
	"time"
)

var baseDurations = []time.Duration{
	05 * time.Minute, 30 * time.Minute, 60 * time.Minute,
	06 * time.Hour, 12 * time.Hour, 24 * time.Hour,
	72 * time.Hour, 168 * time.Hour, 720 * time.Hour,
}

var hourToDayMapping = map[string]string{
	"72h":  "3d",
	"168h": "7d",
	"720h": "30d",
}

func dashboardDurations(maxDuration time.Duration) []time.Duration {
	var result []time.Duration
	for _, td := range baseDurations {
		if td <= maxDuration {
			result = append(result, td)
		}
	}
	return result
}

func shortDur(durations []time.Duration) []string {
	var result []string
	for _, td := range durations {
		s := td.String()
		if strings.HasSuffix(s, "m0s") {
			s = s[:len(s)-2]
		}
		if strings.HasSuffix(s, "h0m") {
			s = s[:len(s)-2]
		}
		if d, ok := hourToDayMapping[s]; ok {
			s = d
		}
		result = append(result, s)
	}
	return result
}
