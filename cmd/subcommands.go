package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/Tarick/tscheduler/job"

	"github.com/spf13/cobra"
)

var (
	// go build -ldflags "-X github.com/Tarick/tscheduler/cmd.buildVer=$(git describe --tags --abbrev=0 HEAD) -X github.com/Tarick/tscheduler/cmd.buildTime=$(date +'%Y-%m-%d_%T')"
	buildVer   string
	buildTime  string
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Print the version number and build date",
		Long:  `Version information and build date`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println("Tscheduler ", buildVer, "-", buildTime)
		},
	}
	runCmd = &cobra.Command{
		Use:   "run",
		Short: "Runs scheduling service",
		Long:  `Runs scheduler, use this to initially start it`,
		Run: func(cmd *cobra.Command, args []string) {
			runScheduler()
		},
	}

	parseCmd = &cobra.Command{
		Use:   "parse",
		Short: "Parse config and print configured jobs",
		Long:  `Parses config to validate it, prints jobs and estimated runs`,
		Run: func(cmd *cobra.Command, args []string) {
			printParsedJobs()
		},
	}
	resumeCmd = &cobra.Command{
		Use:   "resume",
		Short: "Resumes scheduling",
		Long:  `Resumes previously stopped scheduling.`,
		Run: func(cmd *cobra.Command, args []string) {
			startScheduler()
		},
	}
	pauseCmd = &cobra.Command{
		Use:   "stop",
		Short: "Pauses (suspends) scheduling",
		Long:  `Pause suspends scheduling, which can be resumed with "resume".`,
		Run: func(cmd *cobra.Command, args []string) {
			stopScheduler()
		},
	}
	statusCmd = &cobra.Command{
		Use:   "status",
		Short: "Status returns current scheduler and jobs state",
		Long:  `Status returns state of scheduler and prints registered jobs.`,
		Run: func(cmd *cobra.Command, args []string) {
			getStatus()
		},
	}
	shutdownCmd = &cobra.Command{
		Use:   "shutdown",
		Short: "Gracefully stops scheduler and causes program to exit",
		Long:  `Stops scheduler by calling /stop on management interface and make program exit.`,
		Run: func(cmd *cobra.Command, args []string) {
			shutdown()
		},
	}
)

func init() {
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(parseCmd)
	rootCmd.AddCommand(pauseCmd)
	rootCmd.AddCommand(resumeCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(shutdownCmd)
}

func printParsedJobs() {
	runs := 5
	fmt.Printf("Jobs next %d scheduled runs:\n", runs)
	for _, jobConfig := range jobConfigs {
		command := createCommand(jobConfig)
		schedule := []job.Schedule{}
		for _, scheduleSpec := range jobConfig.ScheduleSpec {
			if s, err := job.NewSchedule(scheduleSpec); err != nil {
				fmt.Printf("Failure creating schedule for job %v: %v \n", jobConfig.ID, err)
				os.Exit(1)
			} else {
				schedule = append(schedule, s)
			}
		}
		j, err := job.NewJob(jobConfig.ID, command, schedule)
		if err != nil {
			fmt.Println("Failure creating job: ", err)
			continue
		}
		t := time.Now()
		fmt.Printf("\n %v\n", j)
		for i := 1; i <= runs; i++ {
			t, err = j.Next(t)
			if err != nil {
				fmt.Println(err)
				continue
			}
			fmt.Printf("%d run for the Job: %v\n", i, t)
			t = t.Add(1 * time.Second)
		}
	}
}
