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
	"strings"
	"time"
)

var baseDurations = []time.Duration{
	01 * time.Minute, 05 * time.Minute, 30 * time.Minute, 60 * time.Minute, // Minutes
	06 * time.Hour, 24 * time.Hour, // Hours
	72 * time.Hour, 168 * time.Hour, 720 * time.Hour, // Days
}

func dashboardDurations(maxDuration time.Duration) ([]time.Duration, []string) {
	var result []time.Duration
	for _, td := range baseDurations {
		if td <= maxDuration {
			result = append(result, td)
		}
	}
	if result[len(result)-1] != maxDuration {
		result = append(result, maxDuration)
	}
	return result, shortDur(result)
}

func shortDur(durations []time.Duration) []string {
	var result []string
	for _, td := range durations {
		// Convert days first.
		if td.Truncate(24*time.Hour) == td {
			days := td / (24 * time.Hour)
			if days == 1 {
				result = append(result, "24h")
				continue
			}
			result = append(result, fmt.Sprintf("%dd", days))
			continue
		}

		s := td.String()
		if strings.HasSuffix(s, "m0s") {
			s = s[:len(s)-2]
		}
		if strings.HasSuffix(s, "h0m") {
			s = s[:len(s)-2]
		}
		result = append(result, s)
	}
	return result
}
