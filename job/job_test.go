package job

import (
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

// FIXME: We need custom comparer here
// func TestNewJob(t *testing.T) {
// 	command := func() {}
// 	newScheduleSpecs := []ScheduleSpec{
// 		{Year: "2019, 2021-2022, 2028, 2033",
// 			Month:    "1, 2, 4-5, 3, 4",
// 			Day:      "1",
// 			Weekday:  "0,1,2",
// 			Hour:     "1",
// 			Minute:   "2, 3",
// 			Second:   "3",
// 			Location: "UTC"},
// 		{Year: "*",
// 			Month:   "*",
// 			Day:     "*",
// 			Weekday: "*",
// 			Hour:    "*",
// 			Minute:  "*",
// 			Second:  "0"},
// 	}
// 	expectedJob := Job{
// 		id:       "TestNew",
// 		execFunc: command,
// 		schedule: []Schedule{
// 			{
// 				year:     []int{2019, 2021, 2022, 2028, 2033},
// 				month:    []int{1, 2, 3, 4, 5},
// 				day:      []int{1},
// 				weekday:  []int{0, 1, 2},
// 				hour:     []int{1},
// 				minute:   []int{2, 3},
// 				second:   []int{3},
// 				location: time.UTC,
// 			},
// 			{
// 				year:     []int{},
// 				month:    []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12},
// 				day:      []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31},
// 				weekday:  []int{0, 1, 2, 3, 4, 5, 6},
// 				hour:     []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23},
// 				minute:   []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16, 17, 18, 19, 20, 21, 22, 23, 24, 25, 26, 27, 28, 29, 30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40, 41, 42, 43, 44, 45, 46, 47, 48, 49, 50, 51, 52, 53, 54, 55, 56, 57, 58, 59},
// 				second:   []int{0},
// 				location: time.Local,
// 			},
// 		},
// 	}

// 	schedule := []Schedule{}
// 	for _, scheduleSpec := range newScheduleSpecs {
// 		s, _ := NewSchedule(scheduleSpec)
// 		schedule = append(schedule, s)
// 	}
// 	job, err := NewJob("TestNew", command, schedule)
// 	if err != nil {
// 		t.Errorf("Error creating new job: %v", err)
// 		return
// 	}
// 	// fmt.Printf("Job: \n %v\n", job)
// 	// fmt.Printf("ExpectedJob: \n %v\n", &expectedJob)
// 	if diff := cmp.Diff(job, &expectedJob, cmp.AllowUnexported(Job{}), cmp.AllowUnexported(Schedule{}), cmp.AllowUnexported(time.Location{})); diff != "" {
// 		// if diff := cmp.Diff(job, &expectedJob, DeepAllowUnexported(job, &expectedJob)); diff != "" {
// 		t.Errorf("New Job creation test failed!\nEXPECTED: \n %v\nNEW: \n %v\nDIFF: %v\n",
// 			&expectedJob, job, diff)
// 	}

// }

func TestNext(t *testing.T) {
	newScheduleSpecs := []ScheduleSpec{
		{Year: "2021-2022, 2028, 2033",
			Month:    "1, 2, 4-5, 3, 4",
			Day:      "1",
			Weekday:  "0,1,2",
			Hour:     "1",
			Minute:   "2, 3",
			Second:   "3",
			Location: "UTC"},
		{Year: "*",
			Month:    "*",
			Day:      "*",
			Weekday:  "1,2",
			Hour:     "*",
			Minute:   "*",
			Second:   "0",
			Location: "UTC",
		},
	}
	schedule := []Schedule{}
	for _, scheduleSpec := range newScheduleSpecs {
		s, _ := NewSchedule(scheduleSpec)
		schedule = append(schedule, s)
	}
	job, err := NewJob("TestNew", func() {}, schedule)
	if err != nil {
		t.Errorf("Error creating new job: %v", err)
		return
	}
	currTime := time.Date(2020, 03, 1, 0, 0, 0, 0, time.UTC)
	jobNextRun, _ := job.Next(currTime)
	jobExpectedNextRun := time.Date(2020, 3, 2, 0, 0, 0, 0, time.UTC)
	if diff := cmp.Diff(jobNextRun, jobExpectedNextRun); diff != "" {
		t.Errorf("Test for job next time run failed!\nEXPECTED: \n %v\nNEW: \n %v\nDIFF: %v\n",
			jobExpectedNextRun, jobNextRun, diff)
	}
}
