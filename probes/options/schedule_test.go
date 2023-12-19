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

package options

import (
	"testing"
	"time"

	configpb "github.com/cloudprober/cloudprober/probes/proto"
	"github.com/stretchr/testify/assert"
	"google.golang.org/protobuf/proto"
)

func TestSchedules(t *testing.T) {
	scheduleConfs := []*configpb.Schedule{
		{
			Type:         configpb.Schedule_DISABLE.Enum(),
			StartTime:    proto.String("22:00"),
			StartWeekday: configpb.Schedule_FRIDAY.Enum(),
			EndWeekday:   configpb.Schedule_SUNDAY.Enum(),
			EndTime:      proto.String("16:59"),
			Timezone:     proto.String("America/New_York"),
		},
		{
			Type:         configpb.Schedule_DISABLE.Enum(),
			StartTime:    proto.String("19:00"),
			StartWeekday: configpb.Schedule_TUESDAY.Enum(),
			EndWeekday:   configpb.Schedule_TUESDAY.Enum(),
			EndTime:      proto.String("21:00"),
			Timezone:     proto.String("America/New_York"),
		},
	}

	everyDayDisableConf := []*configpb.Schedule{
		{
			Type:      configpb.Schedule_DISABLE.Enum(),
			StartTime: proto.String("18:00"), // Default end time is 23:59
			Timezone:  proto.String("America/New_York"),
		},
		{
			Type:    configpb.Schedule_DISABLE.Enum(),
			EndTime: proto.String("11:59"), // UTC: NY 06:59
		},
	}

	enableDisable := []*configpb.Schedule{
		{
			Type:         configpb.Schedule_ENABLE.Enum(),
			StartWeekday: configpb.Schedule_MONDAY.Enum(),
			StartTime:    proto.String("08:00"),
			EndWeekday:   configpb.Schedule_FRIDAY.Enum(),
			EndTime:      proto.String("20:00"),
			Timezone:     proto.String("America/New_York"),
		},
		{
			Type:      configpb.Schedule_DISABLE.Enum(),
			StartTime: proto.String("17:00"),
			EndTime:   proto.String("17:59"),
			Timezone:  proto.String("America/New_York"),
		},
	}

	tests := []struct {
		name    string
		confs   []*configpb.Schedule
		results map[string]bool
		wantErr bool
	}{
		{
			name: "err-time-spec",
			confs: []*configpb.Schedule{
				{
					Type:      configpb.Schedule_DISABLE.Enum(),
					StartTime: proto.String("2200"),
					EndTime:   proto.String("2359"),
				},
			},
			wantErr: true,
		},
		{
			name: "err-invalid-sched",
			confs: []*configpb.Schedule{
				{
					Type:      configpb.Schedule_DISABLE.Enum(),
					StartTime: proto.String("22:00"),
					EndTime:   proto.String("16:59"), // End time < start time
				},
			},
			wantErr: true,
		},
		{
			name:  "weekendAndRollout",
			confs: scheduleConfs,
			results: map[string]bool{
				"2023-12-14 22:00:00 -0500": true,  // Thu
				"2023-12-15 21:59:00 -0500": true,  // Fri
				"2023-12-15 22:00:00 -0500": false, // Fri -- disable for the weekend
				"2023-12-16 12:00:00 -0500": false, // Sat
				"2023-12-17 16:59:00 -0500": false, // Sun
				"2023-12-17 17:00:00 -0500": true,  // Sun -- re-enable for Asia
				"2023-12-19 17:00:00 -0500": true,  // Tue
				"2023-12-19 16:30:00 -0800": false, // Tue -- disable for rollouts (PST)
				"2023-12-19 19:30:00 -0500": false, // Tue -- disable for rollouts
				"2023-12-19 21:01:00 -0500": true,  // Tue -- re-enabel after rollout
			},
		},
		{
			name:  "everydayDisable",
			confs: everyDayDisableConf,
			results: map[string]bool{
				"2023-12-14 17:59:00 -0500": true,  // Thu
				"2023-12-14 18:01:00 -0500": false, // Thu
				"2023-12-15 00:01:00 -0500": false, // Fri
				"2023-12-15 05:01:00 -0500": false, // Fri
				"2023-12-15 07:01:00 -0500": true,  // Fri
			},
		},
		{
			name:  "enableDisable",
			confs: enableDisable,
			results: map[string]bool{
				"2023-12-10 17:59:00 -0500": false, // Sun
				"2023-12-11 08:01:00 -0500": true,  // Mon
				"2023-12-11 17:01:00 -0500": false, // Mon
				"2023-12-11 18:01:00 -0500": true,  // Mon
				"2023-12-15 22:01:00 -0500": false, // Fri
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := NewSchedule(tt.confs, nil)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseSchedules() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			for timeStr, want := range tt.results {
				t.Run(timeStr, func(t *testing.T) {
					ttime, _ := time.Parse("2006-01-02 15:04:05 -0700", timeStr)
					assert.Equal(t, want, s.isIn(ttime))
				})
			}
		})
	}
}
