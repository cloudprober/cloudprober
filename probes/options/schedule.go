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

	"github.com/cloudprober/cloudprober/logger"
	configpb "github.com/cloudprober/cloudprober/probes/proto"
)

type period struct {
	baseTime           time.Time
	loc                *time.Location
	everyDay           bool
	startTime, endTime time.Time
	l                  *logger.Logger
}

// normalizeTime is the most tricky part of the schedule implementation. It
// moves the provided time to baseTime's reference frame. Base time is set to
// Jan 1, 2023 00:00:00, which was a Sunday. For example, if provided time is
// Wed 10:00:00, normalizeTime will become "Jan 4, 2023 10:00:00".
//
// Note that normalizeTime doesn't take care of the timezone. It assumes that
// the provided time is in the same timezone as the period.
func (p *period) normalizeTime(weekDay, h, m int) time.Time {
	timeSinceToday := (time.Duration(h) * time.Hour) + (time.Duration(m) * time.Minute)
	if p.everyDay {
		return p.baseTime.Add(timeSinceToday)
	}

	// If it's not everyday, we need to move the base time by the number of
	// given weekdays.
	timeSinceSunday := (time.Duration(weekDay) * (24 * time.Hour)) + timeSinceToday
	return p.baseTime.Add(timeSinceSunday)
}

// isInPeriod returns true if the provided time is in the schedule.
func (p *period) isInPeriod(t time.Time) bool {
	// Convert the provided time to the same timezone as the period.
	t = t.In(p.loc)

	nt := p.normalizeTime(int(t.Weekday()), t.Hour(), t.Minute())

	// For times between Sunday 00:00 and the start of the schedule.
	if !p.everyDay && nt.Before(p.startTime) {
		nt = nt.Add(7 * 24 * time.Hour)
	}

	p.l.Debugf("Schedule: checking if %s is in schedule: %s", nt.String(), p.String())

	if nt == p.startTime || nt == p.endTime {
		return true
	}
	return nt.After(p.startTime) && nt.Before(p.endTime)
}

func parsePeriod(sched *configpb.Schedule, l *logger.Logger) (*period, error) {
	p := &period{l: l}

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
	p.loc = loc
	p.baseTime, _ = time.ParseInLocation("2006-Jan-02", "2023-Jan-01", p.loc)

	startTimeHour, startTimeMin, err := parseTime(sched.GetStartTime())
	if err != nil {
		return nil, fmt.Errorf("error parsing start time (%s): %v", sched.GetStartTime(), err)
	}
	p.startTime = p.normalizeTime(weekDayNum(sched.GetStartWeekday()), startTimeHour, startTimeMin)

	endTimeHour, endTimeMin, err := parseTime(sched.GetEndTime())
	if err != nil {
		return nil, fmt.Errorf("error parsing end time (%s): %v", sched.GetEndTime(), err)
	}
	p.endTime = p.normalizeTime(weekDayNum(sched.GetEndWeekday()), endTimeHour, endTimeMin)

	if p.endTime.Before(p.startTime) {
		if p.everyDay || sched.GetStartWeekday() == sched.GetEndWeekday() {
			return nil, fmt.Errorf("invalid schedule: for same day start time (%s) should be after end time (%s)", sched.GetStartTime(), sched.GetEndTime())
		}
		p.endTime = p.endTime.Add(7 * 24 * time.Hour)
	}

	l.Infof("Schedule: %s", p.String())

	return p, nil
}

func (p *period) String() string {
	if p.everyDay {
		return fmt.Sprintf("Everyday %s - %s", p.startTime.Format("15:04 MST"), p.endTime.Format("15:04 MST"))
	}
	return fmt.Sprintf("%s - %s", p.startTime.Format("Mon 15:04 MST"), p.endTime.Format("Mon 15:04 MST"))
}

type Schedule struct {
	enablePeriods  []*period
	disablePeriods []*period
	l              *logger.Logger
}

// isIn returns true if the probe is enabled at the given time.
func (s *Schedule) isIn(t time.Time) bool {
	if s == nil {
		return true
	}

	isEnabled := false
	if len(s.enablePeriods) == 0 {
		isEnabled = true
	}

	for _, p := range s.enablePeriods {
		if p.isInPeriod(t) {
			s.l.Debugf("Schedule: in enable period: %s", p.String())
			isEnabled = true
			break
		}
	}

	for _, p := range s.disablePeriods {
		if p.isInPeriod(t) {
			s.l.Debugf("Schedule: in disable period: %s", p.String())
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

func NewSchedule(scheds []*configpb.Schedule, l *logger.Logger) (*Schedule, error) {
	s := &Schedule{l: l}
	for _, sched := range scheds {
		p, err := parsePeriod(sched, l)
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
