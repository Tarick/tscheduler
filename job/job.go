package job

import (
	"fmt"
	"sort"
	"time"
)

// MaxYearsAhead is maximum years in future we will try to lookup
const MaxYearsAhead int = 10

// NewJob creates and returns new job struct.
func NewJob(id string, command func(), schedule []Schedule) (*Job, error) {
	j := &Job{}
	j.id, j.execFunc, j.schedule = id, command, schedule
	// Check if we can schedule it at all
	if _, err := j.Next(time.Now()); err != nil {
		return &Job{}, err
	}
	return j, nil
}

// Job definition
type Job struct {
	id       string
	execFunc func()
	schedule []Schedule
	nextRun  time.Time
	lastRun  time.Time
}

// Run starts Job command.
func (j *Job) Run() {
	j.execFunc()
}

// ScheduleSpec defines time patterns to parse.
type ScheduleSpec struct {
	Second, Minute, Hour, Day, Weekday, Month, Year, Location string
}

// Schedule struct is numeric representation of already parsed ScheduleSpec and is used during scheduling
type Schedule struct {
	second, minute, hour, day, weekday, month, year []int
	location                                        *time.Location
}

// NewSchedule creates Schedule with all available variants in numeric form from string based ScheduleSpec
// It sets sane defaults in case any entry is missing, but Minute, Hour, Day, Month must be specified.
// Year by default is * - "every year".
// Second by default is 0.
// Default Timezone (Location) - Local.
func NewSchedule(spec ScheduleSpec) (s Schedule, err error) {
	// Year
	if len(spec.Year) == 0 {
		spec.Year = "*"
	}
	s.year, err = parseYearSpec(spec.Year)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse error of Year %s: %w", spec.Year, err)
	}
	// Month
	if len(spec.Month) == 0 || len(spec.Day) == 0 || len(spec.Hour) == 0 || len(spec.Minute) == 0 {
		return Schedule{}, fmt.Errorf("parse error: schedule months, day, hour and minute must be specified")
	}
	s.month, err = parseTimeSpec(spec.Month, 1, 12)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse error of month spec %s: %w", spec.Month, err)
	}
	// Weekday
	if len(spec.Weekday) == 0 {
		spec.Weekday = "*"
	}
	s.weekday, err = parseTimeSpec(spec.Weekday, 0, 6)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse error of Weekday spec %s: %w", spec.Weekday, err)
	}
	// Day (day of month)
	s.day, err = parseTimeSpec(spec.Day, 1, 31)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse error of Day spec %s: %w", spec.Day, err)
	}
	// Hour
	s.hour, err = parseTimeSpec(spec.Hour, 0, 23)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse error of Hour spec %s: %w", spec.Hour, err)
	}
	// Minute
	s.minute, err = parseTimeSpec(spec.Minute, 0, 59)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse error of Minute spec %s: %w", spec.Minute, err)
	}
	// Second
	if len(spec.Second) == 0 {
		spec.Second = "0"
	}
	s.second, err = parseTimeSpec(spec.Second, 0, 59)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse error of Seconds spec %s: %w", spec.Second, err)
	}
	// Location
	if len(spec.Location) == 0 {
		spec.Location = "Local"
	}
	location, err := time.LoadLocation(spec.Location)
	if err != nil {
		return Schedule{}, fmt.Errorf("parse error of Location spec %s: %w", spec.Location, err)
	}
	s.location = location
	return
}

func (s Schedule) String() string {
	return fmt.Sprintf("\nYears: %v\nMonths: %v\nDays: %v\nWeekdays: %v\nHours: %v\nMinutes: %v\nSeconds: %v\nLocation: %v\n",
		s.year, s.month, s.day, s.weekday, s.hour, s.minute, s.second, s.location)
}
func (s ScheduleSpec) String() string {
	return fmt.Sprintf("\nYears: %v\nMonths: %v\nDays: %v\nWeekdays: %v\nHours: %v\nMinutes: %v\nSeconds: %v\nLocation: %v\n",
		s.Year, s.Month, s.Day, s.Weekday, s.Hour, s.Minute, s.Second, s.Location)
}
func (j *Job) String() string {
	return fmt.Sprintf("ID: %v\nNext run: %v\nLast run: %v", j.id, j.nextRun, j.lastRun)
}

// Updates job nextRun field, which is actually used by Scheduler
func (j *Job) updateNextRun(t time.Time) (err error) {
	t = t.Add(1 * time.Second)
	j.nextRun, err = j.Next(t)
	return err
}

// Next returns the next time for specific Schedule to fire a job and error in case it didn't find the time (e.g. expired Schedule)
func (j *Job) Next(t time.Time) (time.Time, error) {
	nextRuns := []time.Time{}
	var err error
	for _, s := range j.schedule {
		next, err := s.Next(t)
		if err != nil {
			// expired schedule, skip
			continue
		}
		nextRuns = append(nextRuns, next)
	}
	sort.Slice(nextRuns, func(i, j int) bool { return nextRuns[i].Before(nextRuns[j]) })
	if len(nextRuns) == 0 {
		return time.Time{}, fmt.Errorf("no valid next time run found")
	}

	return nextRuns[0], err
}

// Next returns schedule next time to run
func (s *Schedule) Next(t time.Time) (time.Time, error) {
	// TODO: benchmark and optimize this
	t = t.Round(time.Second).In(s.location)
	var err error
	var year, month, day, hour, minute, second, weekday int
	years := s.year
	// Every year
	if len(s.year) == 0 {
		years, _ = makeRange(t.Year(), t.Year()+MaxYearsAhead)
	}
WRAP:
	year = t.Year()

	for i, entry := range years {
		if entry == year {
			break
		}
		if entry > year {
			// reset to 1 Jan new year
			t = time.Date(int(entry), 1, 1, 0, 0, 0, 0, s.location)
			break
		}
		// Couldn't find next date, bail
		if i == len(s.year)-1 {
			return time.Time{}, fmt.Errorf("couldn't find next run time")
		}
	}
	month = int(t.Month())
	for i, entry := range s.month {
		if entry == month {
			break
		}
		if entry > month {
			// reset to 1 Jan new year
			t = time.Date(t.Year(), time.Month(entry), 1, 0, 0, 0, 0, s.location)
			break
		}
		// Jump to New Year
		if i == len(s.month)-1 {
			t = time.Date(t.Year()+1, 1, 1, 0, 0, 0, 0, s.location)
			goto WRAP
		}
	}
	day = t.Day()
	for i, entry := range s.day {
		// fmt.Println("Day:", entry, t)
		if entry == day {
			break
		}
		if entry > day {
			if t.Month() != time.Date(t.Year(), t.Month(), int(entry), 0, 0, 0, 0, s.location).Month() {
				t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, s.location)
				goto WRAP
			}
			t = time.Date(t.Year(), t.Month(), int(entry), 0, 0, 0, 0, s.location)
			break
		}
		// Jump to start of next month
		if i == len(s.day)-1 {
			t = time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, s.location)
			goto WRAP
		}
	}
	weekday = int(t.Weekday())
	for i, entry := range s.weekday {
		if entry == weekday {
			break
		}
		// Jump to start of next day
		if i == len(s.weekday)-1 {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, s.location)
			goto WRAP
		}
	}
	hour = t.Hour()
	for i, entry := range s.hour {
		if entry == hour {
			break
		}
		if entry > hour {
			t = time.Date(t.Year(), t.Month(), t.Day(), int(entry), 0, 0, 0, s.location)
			break
		}
		// Jump to start of next day
		if i == len(s.hour)-1 {
			t = time.Date(t.Year(), t.Month(), t.Day()+1, 0, 0, 0, 0, s.location)
			goto WRAP
		}
	}
	minute = t.Minute()
	for i, entry := range s.minute {
		if entry == minute {
			break
		}
		if entry > minute {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), int(entry), 0, 0, s.location)
			break
		}
		// Jump to start of next hour
		if i == len(s.minute)-1 {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour()+1, 0, 0, 0, s.location)
			goto WRAP
		}
	}
	second = t.Second()
	for i, entry := range s.second {
		if entry == second {
			break
		}
		if entry > second {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), int(entry), 0, s.location)
			break
		}
		// Jump to start of next minute
		if i == len(s.second)-1 {
			t = time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute()+1, 0, 0, s.location)
			goto WRAP
		}
	}
	return t, err
}
