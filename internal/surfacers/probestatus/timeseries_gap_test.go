// Copyright 2022-2026 The Cloudprober Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package probestatus

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestTimeseriesGap(t *testing.T) {
	type datumInput struct {
		minuteOffset int
		success      int64
		total        int64
	}

	type filledRange struct {
		fromIdx, toIdx int
		success        int64
	}

	type agoCheck struct {
		minutes  int
		expected int
	}

	type deltaCheck struct {
		minutesBehind int
		minutes       int
		wantTotal     int64
		wantSuccess   int64
	}

	tests := []struct {
		name      string
		size      int
		data      []datumInput
		latestIdx int
		oldestIdx int
		// For each intermediate range [fromMinute, toMinute), assert that
		// the bucket was filled with the given success value.
		filledRanges []filledRange
		// agoIndex checks: durationMinutes -> expected index
		agoChecks []agoCheck
		// computeDelta checks
		deltaChecks []deltaCheck
	}{
		{
			name: "simple gap no rotation",
			size: 100,
			data: []datumInput{
				{minuteOffset: 0, success: 9, total: 10},
				{minuteOffset: 5, success: 18, total: 20},
			},
			latestIdx: 5,
			oldestIdx: 0,
			filledRanges: []filledRange{
				{fromIdx: 1, toIdx: 4, success: 9}, // buckets 1-4 filled with T=0 datum
			},
			agoChecks: []agoCheck{
				{minutes: 5, expected: 0},
			},
			deltaChecks: []deltaCheck{
				{minutesBehind: 5, minutes: 5, wantTotal: 10, wantSuccess: 9},   // 20-10, 18-9
				{minutesBehind: 3, minutes: 3, wantTotal: 10, wantSuccess: 9},   // 20-10, 18-9 (filled bucket)
				{minutesBehind: 16, minutes: 5, wantTotal: -1, wantSuccess: -1}, // no data for > 5+10 minutes
			},
		},
		{
			name: "two gaps no rotation",
			size: 100,
			data: []datumInput{
				{minuteOffset: 0, success: 9, total: 10},
				{minuteOffset: 5, success: 19, total: 20},
				{minuteOffset: 10, success: 27, total: 30},
			},
			latestIdx: 10,
			oldestIdx: 0,
			filledRanges: []filledRange{
				{fromIdx: 1, toIdx: 4, success: 9},  // filled with T=0 datum
				{fromIdx: 6, toIdx: 9, success: 19}, // filled with T=5 datum
			},
			agoChecks: []agoCheck{
				{minutes: 5, expected: 5},
				{minutes: 10, expected: 0},
			},
			deltaChecks: []deltaCheck{
				{minutes: 5, wantTotal: 10, wantSuccess: 8},   // 30-20, 27-19
				{minutes: 10, wantTotal: 20, wantSuccess: 18}, // 30-10, 27-9
			},
		},
		{
			name: "gap causes rotation",
			size: 10,
			data: []datumInput{
				{minuteOffset: 0, success: 9, total: 10},
				{minuteOffset: 5, success: 18, total: 20},
				// Gap of 8 minutes from T=5 to T=13, with size=10 this
				// will wrap around and push oldestIdx forward.
				{minuteOffset: 13, success: 45, total: 50},
			},
			// latestIdx = (5 + 8) % 10 = 3
			latestIdx: 3,
			// oldestIdx gets pushed as latestIdx wraps past it.
			oldestIdx: 4,
			agoChecks: []agoCheck{
				{minutes: 5, expected: 8},
			},
			deltaChecks: []deltaCheck{
				{minutes: 5, wantTotal: 30, wantSuccess: 27}, // 50-20, 45-18 (filled from T=5)
			},
		},
		{
			name: "gap larger than buffer size",
			size: 10,
			data: []datumInput{
				{minuteOffset: 0, success: 8, total: 10},
				// Gap of 15 minutes, larger than buffer. numMissed clamped
				// to len(ts.a)=10.
				{minuteOffset: 15, success: 95, total: 100},
			},
			latestIdx: 0,
			oldestIdx: 1,
			filledRanges: []filledRange{
				// All intermediate buckets filled with the T=0 datum.
				{fromIdx: 1, toIdx: 9, success: 8},
			},
			deltaChecks: []deltaCheck{
				{minutes: 5, wantTotal: 90, wantSuccess: 87}, // 100-10, 95-8 (filled)
				{minutes: 9, wantTotal: 90, wantSuccess: 87}, // 100-10, 95-8 (all filled same)
			},
		},
		{
			name: "consecutive data then gap with rotation",
			size: 10,
			data: []datumInput{
				{minuteOffset: 0, success: 9, total: 10},
				{minuteOffset: 1, success: 11, total: 12},
				{minuteOffset: 2, success: 13, total: 14},
				{minuteOffset: 3, success: 14, total: 16},
				{minuteOffset: 4, success: 16, total: 18},
				{minuteOffset: 5, success: 18, total: 20},
				{minuteOffset: 6, success: 20, total: 22},
				{minuteOffset: 7, success: 22, total: 24},
				{minuteOffset: 8, success: 24, total: 26},
				// Now latestIdx=8, oldestIdx=0, buffer nearly full.
				// Gap of 5 minutes causes rotation.
				{minuteOffset: 13, success: 36, total: 40},
			},
			// latestIdx = (8 + 5) % 10 = 3
			latestIdx: 3,
			oldestIdx: 4,
			filledRanges: []filledRange{
				// Buckets 9, 0, 1, 2 filled with T=8 datum (success=24).
				{fromIdx: 9, toIdx: 9, success: 24},
				{fromIdx: 0, toIdx: 2, success: 24},
			},
			deltaChecks: []deltaCheck{
				{minutes: 5, wantTotal: 14, wantSuccess: 12}, // 40-26, 36-24 (filled from T=8)
				{minutes: 9, wantTotal: 22, wantSuccess: 20}, // 40-18, 36-16 (T=4, idx 4)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newTimeseries(time.Minute, tt.size, nil)
			start := time.Now().Truncate(time.Minute)
			ts.startTime = start

			for _, d := range tt.data {
				ts.addDatum(start.Add(time.Duration(d.minuteOffset)*time.Minute), &datum{success: d.success, total: d.total})
			}

			assert.Equal(t, tt.latestIdx, ts.latestIdx, "latestIdx")
			assert.Equal(t, tt.oldestIdx, ts.oldestIdx, "oldestIdx")

			for _, fr := range tt.filledRanges {
				for i := fr.fromIdx; i <= fr.toIdx; i++ {
					assert.NotNilf(t, ts.a[i], "bucket %d should not be nil", i)
					if ts.a[i] != nil {
						assert.Equal(t, fr.success, ts.a[i].success,
							fmt.Sprintf("bucket %d success", i))
					}
				}
			}

			for _, ac := range tt.agoChecks {
				idx := ts.agoIndex(ac.minutes)
				assert.Equal(t, ac.expected, idx,
					fmt.Sprintf("agoIndex(%d)", ac.minutes))
			}

			for _, dc := range tt.deltaChecks {
				ts.currentTS = time.Now().Truncate(time.Minute).Add(time.Duration(-dc.minutesBehind) * time.Minute)
				gotTotal, gotSuccess := ts.computeDelta(time.Duration(dc.minutes) * time.Minute)
				assert.Equal(t, dc.wantTotal, gotTotal,
					fmt.Sprintf("computeDelta(%dm) total", dc.minutes))
				assert.Equal(t, dc.wantSuccess, gotSuccess,
					fmt.Sprintf("computeDelta(%dm) success", dc.minutes))
			}
		})
	}
}
