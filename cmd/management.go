package cmd

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"text/template"
	"time"

	"github.com/Tarick/tscheduler/job"
)

func startManagementService() {
	mngMux := http.NewServeMux()
	mngMux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Available commands: /start, /stop, /status, /shutdown")
	})
	// mngMux.Handle("/status", mngStatusHandler)
	mngMux.HandleFunc("/pause", mngPauseHandler)
	mngMux.HandleFunc("/resume", mngResumeHandler)
	mngMux.HandleFunc("/status", mngStatusHandler)
	mngMux.HandleFunc("/shutdown", mngShutdownHandler)
	mngServer := http.Server{
		Addr:         management.Address,
		Handler:      mngMux,
		ReadTimeout:  15 * time.Minute,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	log.Info("Starting management service on http://", management.Address, "/")
	log.Fatal(mngServer.ListenAndServe())
}

func mngPauseHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, mngStopScheduler())
}

// Stops scheduler with the configurable wait period for jobs to stop. Doesn't kill jobs.
func mngStopScheduler() string {
	ctx := sr.Stop()
	ctx, cancel := context.WithTimeout(ctx, time.Duration(management.SchedulerStopTimeout)*time.Second)
	defer cancel()
	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		return fmt.Sprintf("Scheduler stopped with exceeded configured timeout %v seconds, there were left running jobs", management.SchedulerStopTimeout)
	}
	return fmt.Sprintf("Scheduler stopped")
}

func mngResumeHandler(w http.ResponseWriter, r *http.Request) {
	sr.Start()
	fmt.Fprintf(w, "Scheduler started")
}

func mngStatusHandler(w http.ResponseWriter, r *http.Request) {
	type StatusData struct {
		SchedulerIsRunning bool
		Jobs               []job.Job
	}
	st := StatusData{}
	st.SchedulerIsRunning = sr.IsRunning()
	st.Jobs = sr.GetJobs()

	t := template.New("status")
	t.Parse(`Scheduler is running: {{.SchedulerIsRunning}}
Jobs registerd on scheduler and their stats:
{{ range .Jobs }}
{{ . }} 

{{ end }} `)
	t.Execute(w, st)
}

// Shutdown a scheduler, the process will exit. Kills jobs after the configurable termination period.
func mngShutdownHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, mngStopScheduler())
	log.Warn("Sending global shutdown signal to any available job")
	fmt.Fprintln(w, "Sent message to any still running job to be killed.")
	fmt.Fprintln(w, "Terminating!!!")
	close(jobsCancel)
	log.Debug("Sleeping for ", management.JobsTerminationTimeout, "s to allow jobs to be killed.")
	time.Sleep(time.Duration(management.JobsTerminationTimeout) * time.Second)
	// Send termination to main thread
	done <- struct{}{}
}

func callManagementEndpoint(endpoint string) string {
	if !management.Enabled {
		fmt.Println("ERROR - management is not enabled in config. Stopping and starting won't work without management enabled and scheduler started with it.")
		os.Exit(1)
	}
	managementURL := "http://" + management.Address + endpoint
	resp, err := http.Get(managementURL)
	if err != nil {
		fmt.Printf("ERROR - failed query of management url: %v", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	return string(body)
}

func startScheduler() {
	fmt.Println(callManagementEndpoint("/start"))
}
func stopScheduler() {
	fmt.Println(callManagementEndpoint("/stop"))
}
func getStatus() {
	fmt.Println(callManagementEndpoint("/status"))
}
func shutdown() {
	fmt.Println(callManagementEndpoint("/shutdown"))
}
