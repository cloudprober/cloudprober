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
	"fmt"
	"time"

	_ "time/tzdata"

	configpb "github.com/cloudprober/cloudprober/probes/proto"
)

// baseTime is the time used as a base for all the schedules. It's set to
// 72 hours after the Unix epoch to bring time to Sunday 00:00:00 UTC.
var baseTime = time.Unix(0, 0).Add(72 * time.Hour)

type period struct {
	everyDay           bool
	startTime, endTime time.Time
}

func (p *period) normalizeTime(weekDay, h, m int) time.Time {
	timeSinceToday := (time.Duration(h) * time.Hour) + (time.Duration(m) * time.Minute)
	if p.everyDay {
		return baseTime.Add(timeSinceToday)
	}
	timeSinceSunday := (time.Duration(weekDay) * (24 * time.Hour)) + timeSinceToday
	return baseTime.Add(timeSinceSunday)
}

// isInPeriod returns true if the provided time is in the schedule.
func (p *period) isInPeriod(t time.Time) bool {
	t = t.UTC()
	nt := p.normalizeTime(int(t.Weekday()), t.Hour(), t.Minute())
	if nt.Before(p.startTime) {
		nt = nt.Add(7 * 24 * time.Hour)
	}

	if nt == p.startTime || nt == p.endTime {
		return true
	}
	return nt.After(p.startTime) && nt.Before(p.endTime)
}

type Schedule struct {
	enablePeriods  []*period
	disablePeriods []*period
}

// IsEnabled returns true if the probe is enabled at the given time.
func (s *Schedule) IsEnabled(t time.Time) bool {
	isEnabled := false
	if len(s.enablePeriods) == 0 {
		isEnabled = true
	}

	for _, p := range s.enablePeriods {
		if p.isInPeriod(t) {
			isEnabled = true
			break
		}
	}

	for _, p := range s.disablePeriods {
		if p.isInPeriod(t) {
			isEnabled = false
			break
		}
	}

	return isEnabled
}

func weekDayNum(wd configpb.Schedule_Weekday) int {
	return map[configpb.Schedule_Weekday]int{
		configpb.Schedule_EVERYDAY:  -1,
		configpb.Schedule_SUNDAY:    0,
		configpb.Schedule_MONDAY:    1,
		configpb.Schedule_TUESDAY:   2,
		configpb.Schedule_WEDNESDAY: 3,
		configpb.Schedule_THURSDAY:  4,
		configpb.Schedule_FRIDAY:    5,
		configpb.Schedule_SATURDAY:  6,
	}[wd]
}

func parseTime(t string) (int, int, error) {
	var h, m int
	_, err := fmt.Sscanf(t, "%d:%d", &h, &m)
	if err != nil {
		return 0, 0, err
	}
	return h, m, nil
}

func parsePeriod(sched *configpb.Schedule) (*period, error) {
	p := &period{}

	if sched.GetStartWeekday() == configpb.Schedule_EVERYDAY || sched.GetEndWeekday() == configpb.Schedule_EVERYDAY {
		if sched.GetStartWeekday() != sched.GetEndWeekday() {
			return nil, fmt.Errorf("invalid schedule: if start_weekday is set to EVERYDAY, end_weekday should also be set to EVERYDAY, and vice versa")
		}
	}
	p.everyDay = sched.GetStartWeekday() == configpb.Schedule_EVERYDAY

	loc, err := time.LoadLocation(sched.GetTimezone())
	if err != nil {
		return nil, fmt.Errorf("error loading timezone (%s): %v", sched.GetTimezone(), err)
	}
	_, offset := time.Now().In(loc).Zone()
	offsetDur := time.Duration(offset) * time.Second

	startTimeHour, startTimeMin, err := parseTime(sched.GetStartTime())
	if err != nil {
		return nil, fmt.Errorf("error parsing start time (%s): %v", sched.GetStartTime(), err)
	}
	p.startTime = p.normalizeTime(weekDayNum(sched.GetStartWeekday()), startTimeHour, startTimeMin).Add(-offsetDur)

	endTimeHour, endTimeMin, err := parseTime(sched.GetEndTime())
	if err != nil {
		return nil, fmt.Errorf("error parsing end time (%s): %v", sched.GetEndTime(), err)
	}
	p.endTime = p.normalizeTime(weekDayNum(sched.GetEndWeekday()), endTimeHour, endTimeMin).Add(-offsetDur)

	if p.endTime.Before(p.startTime) {
		p.endTime = p.endTime.Add(7 * 24 * time.Hour)
	}

	return p, nil
}

func ParseSchedules(scheds []*configpb.Schedule) (*Schedule, error) {
	s := &Schedule{}
	for _, sched := range scheds {
		p, err := parsePeriod(sched)
		if err != nil {
			return nil, err
		}

		switch sched.GetType() {
		case configpb.Schedule_ENABLE:
			s.enablePeriods = append(s.enablePeriods, p)
		case configpb.Schedule_DISABLE:
			s.disablePeriods = append(s.disablePeriods, p)
		default:
			return nil, fmt.Errorf("unknown schedule type: %v", sched.GetType())
		}
	}
	return s, nil
}
