package cmd

import (

	// "time"

	"fmt"
	"os"
	"os/exec"
	"time"

	"github.com/Tarick/tscheduler/pkg/job"
	"github.com/Tarick/tscheduler/pkg/scheduler"

	"golang.org/x/sync/semaphore"
)

var (
	sr         *scheduler.Scheduler
	done       chan struct{}
	jobsCancel chan struct{}
)

// This creates command to run
func createCommand(jobConfig jobConfig) func() {
	var parallelSemaphore *semaphore.Weighted
	var timeout time.Duration
	var err error

	// Ensure that we hold the semaphore if job is not parallel
	if !jobConfig.Parallel {
		parallelSemaphore = semaphore.NewWeighted(1)
	}
	if jobConfig.Timeout != "" {
		timeout, err = time.ParseDuration(jobConfig.Timeout)
		if err != nil {
			log.Fatal("Failure parsing timeout value %s: %v", jobConfig.Timeout, err)
		}
	} else {
		// Zero timeout means no timeout
		timeout = 0 * time.Second
	}
	// This func is returned as job.command to be run
	return func() {
		var err error
		// jobStatus is for metrics population
		var jobStatus string
		if metrics.Enabled {
			metricJobsRunning.Inc()
			defer metricJobsRunning.Dec()
			defer func(status *string) {
				metricJobsFinished.WithLabelValues(jobConfig.ID, *status).Inc()
			}(&jobStatus)
		}
		cmd := exec.Command(jobConfig.Command[0], jobConfig.Command[1:]...)
		if jobConfig.Stdout != "" {
			cmd.Stdout, err = os.OpenFile(jobConfig.Stdout, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				log.Error(jobConfig.ID, " failure opening stdout file for writing: ", err)
				jobStatus = "skipped"
				return
			}
		}
		if jobConfig.Stderr != "" {
			cmd.Stderr, err = os.OpenFile(jobConfig.Stderr, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0644)
			if err != nil {
				log.Error(jobConfig.ID, " failure opening stderr file for writing: ", err)
				jobStatus = "skipped"
				return
			}
		}
		// If parallelSemaphore is not nil, then this job is NOT allowed to run simultaneously
		if parallelSemaphore != nil {
			if !parallelSemaphore.TryAcquire(1) {
				log.Warn("Job ", jobConfig.ID, " is already running and its parallel run is disabled, skipped execution.")
				jobStatus = "skipped"
				return
			}
			defer parallelSemaphore.Release(1)
		}
		// Goroutine that actually starts command
		done := make(chan error, 1)
		go func() {
			defer close(done)
			log.Debug("Starting ", jobConfig.ID, " command: ", jobConfig.Command)
			err = cmd.Start()
			if err != nil {
				done <- fmt.Errorf(jobConfig.ID, " command start failed: ", err)
				return
			}
			log.Debug(jobConfig.ID, " - waiting for process to finish: ", jobConfig.Command)
			// Success or error from command if any passed to caller
			done <- cmd.Wait()
		}()
		// if timeout is present, wait for job to exit or kill
		// TODO: can we optimize with context here? too much boilerplate
		if timeout != 0 {
			select {
			case <-time.After(timeout):
				err = cmd.Process.Kill()
				if err != nil {
					err = fmt.Errorf("job was killed as has reached job timeout, but failed to kill process: %v", err)
				} else {
					err = fmt.Errorf("job was killed as has reached job timeout")
				}
			// Wait for global termination signal (channel close), when shutting down the program
			case <-jobsCancel:
				log.Debug("Job ", jobConfig.ID, " received global jobs shutdown signal, terminating")
				err = cmd.Process.Kill()
				if err != nil {
					err = fmt.Errorf("received global shutdown signal, but failed to terminate process: %v", err)
				} else {
					err = fmt.Errorf("received global shutdown signal, process was killed")
				}
			case err = <-done:
			}
		} else {
			select {
			// Wait for global termination signal (channel close), when shutting down the program
			case <-jobsCancel:
				log.Warn("Job ", jobConfig.ID, " received global jobs shutdown signal, terminating")
				err = cmd.Process.Kill()
				if err != nil {
					err = fmt.Errorf("received global shutdown signal, but failed to terminate process: %v", err)
				} else {
					err = fmt.Errorf("received global shutdown signal, process was killed")
				}
			case err = <-done:
			}
		}
		// Finally check the status of job
		if err != nil {
			jobStatus = "failed"
			log.Error(jobConfig.ID, " failed with error: ", err)
		} else {
			// log.Debug(j.id, " job output: ", string(output))
			jobStatus = "success"
			log.Info(jobConfig.ID, " job finished succesfully")
		}
	}
}

func runScheduler() {
	// For this thread
	done = make(chan struct{})
	// For all jobs
	jobsCancel = make(chan struct{})

	sr = scheduler.New(log)
	for _, jobConfig := range jobConfigs {
		schedule := []job.Schedule{}
		for _, scheduleSpec := range jobConfig.ScheduleSpec {
			if s, err := job.NewSchedule(scheduleSpec); err != nil {
				log.Fatalf("Failure parsing schedule for job %v: %v", jobConfig.ID, err)
			} else {
				schedule = append(schedule, s)
			}
		}
		command := createCommand(jobConfig)
		j, err := job.New(jobConfig.ID, command, schedule)
		if err != nil {
			log.Fatalf("Failure creating new job %+v\nError: %v", j, err)
		}
		if err := sr.AddJob(j); err != nil {
			log.Fatalf("Failure adding new Job: %v", err)
		}
	}

	// Start metrics collector and API
	if metrics.Enabled {
		go startMetricsService()
	}
	if management.Enabled {
		// Start into background
		go startManagementService()
	}

	// go sr.StartAndServe()
	sr.Start()
	<-done
	// Give some time for jobs to exit
	time.Sleep(1 * time.Second)
	// time.Sleep(5 * time.Second)
	// if err := sr.RemoveJob("Test Job #2"); err != nil {
	// 	log.Error("Failure removing job: ", err)
	// } else {
	// 	log.Info("Removed job")
	// }
	// log.Info(sr.GetJobs())
	// time.Sleep(5 * time.Second)
	// jb := sr.Jobs[1]
	// if err := sr.AddJob(jb); err != nil {
	// 	log.Error("Failure adding job: ", err)
	// }
	// time.Sleep(5 * time.Second)
	// time.Sleep(5 * time.Second)
	// // sr.RemoveJob(jb.(name))
	// // time.Sleep(5 * time.Second)
	// // sr.AddJob(jb)
	// time.Sleep(5 * time.Second)
	// // log.Debug("Stopping")
	// // sr.Stop()

	log.Info("Stopped")
}
