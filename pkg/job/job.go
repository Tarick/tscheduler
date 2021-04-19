package job

import (
	"fmt"
	"sort"
	"time"
)

// New creates and returns new job struct.
func New(id string, command func(), schedule []Schedule) (*Job, error) {
	j := &Job{}
	j.Id, j.execFunc, j.schedule = id, command, schedule
	// Check if we can schedule it at all
	if _, err := j.Next(time.Now()); err != nil {
		return &Job{}, err
	}
	return j, nil
}

// Job definition
type Job struct {
	Id       string
	execFunc func()
	schedule []Schedule
	nextRun  time.Time
	lastRun  time.Time
}

// Run starts Job command.
func (j *Job) Run() {
	j.execFunc()
}

func (j *Job) String() string {
	return fmt.Sprintf("ID: %v\nNext run: %v\nLast run: %v", j.Id, j.nextRun, j.lastRun)
}

// Updates job nextRun field, which is actually used by Scheduler
func (j *Job) UpdateNextRun(t time.Time) (err error) {
	t = t.Add(1 * time.Second)
	j.nextRun, err = j.Next(t)
	return err
}

// timepoints implements sort.Interface, which doesn't use reflection as sort.Slice does
type timepoints []time.Time

func (t timepoints) Len() int           { return len(t) }
func (t timepoints) Swap(i, j int)      { t[i], t[j] = t[j], t[i] }
func (t timepoints) Less(i, j int) bool { return t[i].Before(t[j]) }

// Next returns the next time for job to start and error in case it didn't find the time (e.g. all Schedules expired)
func (j *Job) Next(t time.Time) (time.Time, error) {
	// nextRuns := []time.Time{}
	var nextRuns timepoints
	for _, s := range j.schedule {
		next, err := s.Next(t)
		if err != nil {
			// expired schedule, skip
			continue
		}
		nextRuns = append(nextRuns, next)
	}
	sort.Sort(nextRuns)
	if len(nextRuns) == 0 {
		return time.Time{}, fmt.Errorf("no valid next time run found")
	}

	return nextRuns[0], nil
}
