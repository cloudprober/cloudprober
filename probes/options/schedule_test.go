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
	"strings"
	"testing"
	"time"
	_ "time/tzdata"

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

	everydayDisableConf := []*configpb.Schedule{
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

	everyWeekdayDisableConf := []*configpb.Schedule{
		{
			Type:      configpb.Schedule_ENABLE.Enum(),
			StartTime: proto.String("07:00"),
			EndTime:   proto.String("18:00"),
			Timezone:  proto.String("America/New_York"),
		},
		{
			Type:         configpb.Schedule_DISABLE.Enum(),
			StartWeekday: configpb.Schedule_SATURDAY.Enum(),
			EndWeekday:   configpb.Schedule_SUNDAY.Enum(),
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
				"2023-12-17 00:59:00 -0500": false, // Sun
				"2023-12-17 17:00:00 -0500": true,  // Sun -- re-enable for Asia
				"2023-12-19 17:00:00 -0500": true,  // Tue
				"2023-12-19 16:30:00 -0800": false, // Tue -- disable for rollouts (PST)
				"2023-12-19 19:30:00 -0500": false, // Tue -- disable for rollouts
				"2023-12-19 21:01:00 -0500": true,  // Tue -- re-enabel after rollout
			},
		},
		{
			name:  "everydayDisable",
			confs: everydayDisableConf[:1],
			results: map[string]bool{
				"2023-12-14 17:59:00 -0500": true,  // Thu
				"2023-12-14 18:01:00 -0500": false, // Thu
				"2023-12-14 20:01:00 -0500": false, // Thu
				"2023-12-15 00:01:00 -0500": true,  // Fri
			},
		},
		{
			name:  "everydayDisableFull",
			confs: everydayDisableConf,
			results: map[string]bool{
				"2023-12-14 17:59:00 -0500": true,  // Thu
				"2023-12-14 18:01:00 -0500": false, // Thu
				"2023-12-14 20:01:00 -0500": false, // Thu
				"2023-12-15 00:01:00 -0500": false, // Fri
				"2023-12-15 05:01:00 -0500": false, // Fri
				"2023-12-15 07:01:00 -0500": true,  // Fri
			},
		},
		{
			name:  "everyWeekdayDisableFull",
			confs: everyWeekdayDisableConf,
			results: map[string]bool{
				"2023-12-14 17:59:00 -0500": true,  // Thu
				"2023-12-14 18:01:00 -0500": false, // Thu
				"2023-12-14 20:01:00 -0500": false, // Thu
				"2023-12-16 00:01:00 -0500": false, // Sat
				"2023-12-16 07:01:00 -0500": false, // Sat - down
				"2023-12-17 07:01:00 -0500": false, // Sun
				"2023-12-18 07:01:00 -0500": true,  // Mon - back
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

func TestPeriodNormalizeTime(t *testing.T) {
	tests := []struct {
		name      string
		everyDay  bool
		wantTable map[string]string
	}{
		{
			name:     "everyDay",
			everyDay: true,
			wantTable: map[string]string{
				"2023-12-14 17:59:00": "2023-01-01 17:59:00", // Thu (4)
				"2023-12-17 18:01:00": "2023-01-01 18:01:00", // Sun (0)
			},
		},
		{
			name: "weekDay",
			wantTable: map[string]string{
				"2023-12-14 17:59:00": "2023-01-05 17:59:00", // Thu (4)
				"2023-12-14 00:00:00": "2023-01-05 00:00:00", // Thu (4)
				"2023-12-17 18:01:00": "2023-01-01 18:01:00", // Sun (0)
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &period{
				baseTime: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				everyDay: tt.everyDay,
			}
			for in, want := range tt.wantTable {
				t.Run(in, func(t *testing.T) {
					ttime, _ := time.Parse("2006-01-02 15:04:05", in)
					wantTime, _ := time.Parse("2006-01-02 15:04:05", want)
					assert.Equal(t, wantTime, p.normalizeTime(int(ttime.Weekday()), ttime.Hour(), ttime.Minute()))
				})
			}
		})
	}
}

func parseTestPeriodSpec(t *testing.T, in string, sched *configpb.Schedule) {
	toks := strings.Split(in, " ")

	if len(toks) == 2 {
		sched.StartTime = proto.String(toks[0])
		sched.EndTime = proto.String(toks[1])
		return
	}

	if len(toks) == 4 {
		sched.StartWeekday = configpb.Schedule_Weekday(configpb.Schedule_Weekday_value[toks[0]]).Enum()
		sched.StartTime = proto.String(toks[1])
		sched.EndWeekday = configpb.Schedule_Weekday(configpb.Schedule_Weekday_value[toks[2]]).Enum()
		sched.EndTime = proto.String(toks[3])
		return
	}
}

func TestParsePeriod(t *testing.T) {
	tests := []struct {
		name          string
		in            string
		wantStartTime string
		wantEndTime   string
		wantEveryDay  bool
		wantErr       bool
	}{
		{
			name:          "everyday",
			in:            "18:00 19:00",
			wantStartTime: "2023-01-01 18:00:00",
			wantEndTime:   "2023-01-01 19:00:00",
			wantEveryDay:  true,
		},
		{
			name:    "everyday-error",
			in:      "18:00 16:00",
			wantErr: true,
		},
		{
			name:    "sameday-error",
			in:      "TUESDAY 18:00 TUESDAY 16:00",
			wantErr: true,
		},
		{
			name:          "sameday",
			in:            "TUESDAY 18:00 TUESDAY 19:00",
			wantStartTime: "2023-01-03 18:00:00",
			wantEndTime:   "2023-01-03 19:00:00",
		},
		{
			name:          "weekdays",
			in:            "MONDAY 08:00 FRIDAY 19:00",
			wantStartTime: "2023-01-02 08:00:00",
			wantEndTime:   "2023-01-06 19:00:00",
		},
		{
			name:          "weekEnd",
			in:            "FRIDAY 21:00 SUNDAY 17:00",
			wantStartTime: "2023-01-06 21:00:00",
			wantEndTime:   "2023-01-08 17:00:00",
			// Sunday gets normalized to next week
		},
	}
	for _, tt := range tests {
		for _, loc := range []string{"UTC", "America/New_York", "America/Los_Angeles", "Asia/Kolkata"} {
			t.Run(tt.name+loc, func(t *testing.T) {
				sched := &configpb.Schedule{
					Timezone: proto.String(loc),
				}
				parseTestPeriodSpec(t, tt.in, sched)

				got, err := parsePeriod(sched, nil)
				if (err != nil) != tt.wantErr {
					t.Errorf("parsePeriod() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
				if err != nil {
					return
				}

				wantStartTime, _ := time.ParseInLocation("2006-01-02 15:04:05", tt.wantStartTime, got.loc)
				wantEndTime, _ := time.ParseInLocation("2006-01-02 15:04:05", tt.wantEndTime, got.loc)

				assert.Equal(t, wantStartTime, got.startTime, "startTime")
				assert.Equal(t, wantEndTime, got.endTime, "endTime")
				assert.Equal(t, tt.wantEveryDay, got.everyDay, "everyDay")
			})
		}
	}
}

func TestIsInPeriod(t *testing.T) {
	tests := []struct {
		name       string
		tz         string
		periodSpec string
		results    map[string]bool
	}{
		{
			name:       "everyday",
			tz:         "America/New_York",
			periodSpec: "18:00 19:00",
			results: map[string]bool{
				"2023-12-14 17:59:00 -0500": false, // Thu
				"2023-12-14 18:01:00 -0500": true,  // Thu
				"2023-12-14 20:01:00 -0500": false, // Thu
				"2023-12-15 00:01:00 -0500": false, // Fri
				"2023-12-17 17:55:00 -0500": false, // Sun
				"2023-12-17 18:55:00 -0500": true,  // Sun
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sched := &configpb.Schedule{
				Timezone: proto.String(tt.tz),
			}
			parseTestPeriodSpec(t, tt.periodSpec, sched)
			p, _ := parsePeriod(sched, nil)

			for timeStr, want := range tt.results {
				ttime, _ := time.Parse("2006-01-02 15:04:05 -0700", timeStr)
				// Use the provided time in multiple timezones
				for _, loc := range []string{"UTC", "America/New_York", "America/Los_Angeles", "Asia/Kolkata"} {
					t.Run(timeStr+loc, func(t *testing.T) {
						tloc, _ := time.LoadLocation(loc)
						ttime = ttime.In(tloc)
						assert.Equal(t, want, p.isInPeriod(ttime))
					})
				}
			}
		})
	}
}
