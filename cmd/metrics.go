package cmd

import (
	"net/http"
	"time"

	"github.com/Tarick/tscheduler/job"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	metricJobsRunning = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "tscheduler_jobs_running",
		Help: "Number of jobs that are in progress.",
	})
	metricJobsFinished = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tscheduler_jobs_finished_by_jobid_status",
			Help: "Total number of jobs invocations with status and job_id labels.",
		},
		[]string{"job_id", "status"},
	)
	metricJobsCountRegisteredDesc = prometheus.NewDesc(
		"tscheduler_jobs_registered",
		"Number of jobs that are registered in scheduler for processing.",
		nil, nil,
	)
)

// SchedulerCollector implements the Collector interface.
type SchedulerCollector struct {
	Scheduler *job.Scheduler
}

// Describe implements Prometheus Collector
func (sc SchedulerCollector) Describe(ch chan<- *prometheus.Desc) {
	prometheus.DescribeByCollect(sc, ch)
}

// Collect first get all jobs using Scheduler methodst
//  Then it creates constant metrics on the fly based on the returned data.
//
// Collect could be called concurrently, we need to make Scheduler methods concurrently safe
func (sc SchedulerCollector) Collect(ch chan<- prometheus.Metric) {
	jobs := sc.Scheduler.GetJobs()
	numberOfRegisteredJobs := len(jobs)
	// oomCountByHost, ramUsageByHost := cc.ClusterManager.ReallyExpensiveAssessmentOfTheSystemState()
	// for host, oomCount := range oomCountByHost {
	ch <- prometheus.MustNewConstMetric(
		metricJobsCountRegisteredDesc,
		prometheus.GaugeValue,
		float64(numberOfRegisteredJobs),
	)
}

func startMetricsService() {
	// Add the standard process and Go metrics to the custom registry.
	// reg.MustRegister(
	// prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
	// )
	prometheus.MustRegister(SchedulerCollector{Scheduler: sr})
	prometheus.MustRegister(metricJobsFinished)
	prometheus.MustRegister(metricJobsRunning)
	metricsMux := http.NewServeMux()
	metricsServer := http.Server{
		Addr:         metrics.Address,
		Handler:      metricsMux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  15 * time.Second,
	}
	metricsMux.Handle("/metrics", promhttp.Handler())
	log.Info("Starting metrics on http://", metrics.Address, "/metrics")
	log.Fatal(metricsServer.ListenAndServe())
}
