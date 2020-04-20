package job

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type logger interface {
	Error(...interface{})
	Info(...interface{})
	Debug(...interface{})
	Warn(...interface{})
}

// Scheduler is main type for running jobs
type Scheduler struct {
	Jobs      []*Job
	logger    logger
	running   bool
	mutex     sync.Mutex
	stop      chan struct{}
	execF     chan func()
	jobWaiter sync.WaitGroup
}

// GetJobs returns the list of jobs, registered in scheduler
func (sr *Scheduler) GetJobs() (jobs []Job) {
	if sr.running {
		commCh := make(chan struct{})
		sr.execF <- func() {
			jobs = sr.getJobs()
			commCh <- struct{}{}
			close(commCh)
		}
		// block until functions exists
		<-commCh
		return
	}
	// if scheduler is not running, just get the list
	return sr.getJobs()
}

// IsRunning returns scheduler running or not
func (sr *Scheduler) IsRunning() bool {
	return sr.running
}

// getJobs returns the list of Jobs, registered in scheduler
func (sr *Scheduler) getJobs() []Job {
	var jobs = make([]Job, len(sr.Jobs))
	for i, j := range sr.Jobs {
		jobs[i] = *j
	}
	return jobs
}

// AddJob is the wrapper to add a job, updates NextRun field in process.
// If a scheduler is already running, the job is added with custom function, passed to scheduler goroutine.
func (sr *Scheduler) AddJob(j *Job) error {
	if sr.running {
		commCh := make(chan struct{})
		var err error
		sr.execF <- func() {
			j := j
			now := time.Now()
			err = j.updateNextRun(now)
			if err != nil {
				err = fmt.Errorf("Job %v is not schedulable: %v", j.id, err)
			} else {
				err = sr.addJob(j)
			}
			close(commCh)
		}
		// block until functions exists
		<-commCh
		return err
	}
	// if scheduler is not running, just add to the list
	return sr.addJob(j)
}

// addJob adds job to the list, returns error on duplicate job name
func (sr *Scheduler) addJob(j *Job) error {
	for _, je := range sr.Jobs {
		if je.id == j.id {
			return fmt.Errorf("%v job already exists", j.id)
		}
	}
	sr.Jobs = append(sr.Jobs, j)
	return nil
}

// RemoveJob is the wrapper to remove job, returns error if job is not present
// If a scheduler is already running, the job is removed with custom function, passed to scheduler goroutine.
func (sr *Scheduler) RemoveJob(name string) error {
	// For running scheduler
	if sr.running {
		commCh := make(chan struct{})
		var err error
		sr.execF <- func() {
			err = sr.removeJob(name)
			close(commCh)
		}
		// read error and return to caller
		<-commCh
		return err
	}
	return sr.removeJob(name)
}

// RemoveJob removes job from Scheduler list. Returns error if nothing is removed.
func (sr *Scheduler) removeJob(id string) error {
	jobs := []*Job{}
	for _, j := range sr.Jobs {
		if j.id == id {
			continue
		}
		jobs = append(jobs, j)
	}
	if len(jobs) == len(sr.Jobs) {
		return fmt.Errorf("%v job does not exist", id)
	}
	sr.Jobs = jobs
	return nil
}

// NewScheduler constructs scheduler
func NewScheduler(l logger) *Scheduler {
	return &Scheduler{
		logger: l,
		stop:   make(chan struct{}),
		execF:  make(chan func()),
	}
}

// Stop stops the scheduler scheduler, caller gets context to either wait for running jobs to finish or cancel and exit.
func (sr *Scheduler) Stop() context.Context {
	if sr.running {
		sr.stop <- struct{}{}
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		sr.jobWaiter.Wait()
		cancel()
	}()
	return ctx
}

// starts job and adds to the wait list
func (sr *Scheduler) startJob(j *Job) {
	sr.jobWaiter.Add(1)
	go func() {
		defer sr.jobWaiter.Done()
		j.Run()
	}()
}

// Start scheduler asynchronously
func (sr *Scheduler) Start() {
	go sr.StartAndServe()
}

// StartAndServe starsts scheduler with preliminary running check
func (sr *Scheduler) StartAndServe() {
	if sr.running {
		sr.logger.Error("Scheduler is already running, not starting new")
		return
	}
	sr.mutex.Lock()
	defer sr.mutex.Unlock()
	sr.running = true
	sr.start()
}

// Actual scheduler start
func (sr *Scheduler) start() {
	log := sr.logger
	log.Info("Started scheduler")
	now := time.Now()
	for _, j := range sr.Jobs {
		if err := j.updateNextRun(now); err != nil {
			log.Warn("Job \"", j.id, "\" will not be scheduled due to error: ", err)
		}
	}
	var timer *time.Timer
	for {
		wakeUpAt, err := sr.jobsNextRun(now)
		if err != nil {
			log.Warn("Can't schedule next wakeup, ", err)
		}
		log.Debug("Next wake up at: ", wakeUpAt)
		timer = time.NewTimer(wakeUpAt.Sub(now))
		select {
		case now = <-timer.C:
			sr.logger.Debug("Scheduler woke up")
			for _, j := range sr.Jobs {
				if j.nextRun.IsZero() {
					continue
				}
				if j.nextRun.Before(now) {
					j.lastRun = now
					log.Info("Starting job ", j.id, ", scheduled at: ", j.nextRun, ", current time: ", j.lastRun)
					sr.startJob(j)
					if err := j.updateNextRun(now); err != nil {
						log.Warn("Job \"", j.id, "\" will not be scheduled further due to scheduling error: ", err)
					} else {
						log.Info("Job \"", j.id, "\" next run scheduled at: ", j.nextRun)
					}
				}
			}
		case <-sr.stop:
			timer.Stop()
			sr.logger.Info("Scheduler stopped")
			sr.running = false
			return
		// execute any function, passed to the scheduler
		case f := <-sr.execF:
			timer.Stop()
			f()
			now = time.Now()
		}
	}

}

// Updates all jobs NextRun field
func (sr *Scheduler) jobsNextRun(t time.Time) (wakeUp time.Time, err error) {
	// Create initial wakeup value that is too far in the future (10 years).
	initialWakeUp := t.AddDate(10, 0, 0)
	wakeUp = initialWakeUp

	for _, j := range sr.Jobs {
		if j.nextRun.After(t) && j.nextRun.Before(wakeUp) {
			wakeUp = j.nextRun
			err = nil
		}
	}
	if wakeUp.Equal(initialWakeUp) {
		return wakeUp, fmt.Errorf("couldn't find next sleep time for jobs:\n %v", sr.Jobs)
	}
	return wakeUp, nil
}
