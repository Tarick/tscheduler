# tscheduler
Command line tasks/jobs scheduler with seconds to years precision and jobs control

## Disclaimer

This is early ALPHA and will probably stay as it is, you've been warned.

If you're looking for Cron compatible implementation to use as a library in your applications, then please look at [robfig/cron](https://github.com/robfig/cron) or any of __gron__ forks, I've used a lot of ideas from those.

I don't think that __tscheduler__ is mature enough to be used as a separate library package, so will not split it into the library from application code for now.

## Features

* Multiple schedules per job to run the same command on different days at different times.
This is not [Cron expressions](https://en.wikipedia.org/wiki/Cron#CRON_expression) compatible implementation, I've struggled a lot trying to recall the position of specific time in its spec, so tscheduler was written to use fully specified time keywords in its config instead. It still uses some of Crontab constructs to specify schedule for convenience:

  * "*" - all time spec values, i.e. for minutes: 00 - 59.

  * "start/step" ,e.g. */4 for 0, 4, ..., 56. Or 40/6 for minute to specify 40, 46, 52, 58, *00* - BUT BEWARE, we end on 00, NOT 58min + 6min = 4min.

  * "min-max" ranges, e.g. 40-50 means 40, 41, 42, ... , 50.
  * comma separated values, e.g. 4, 5, 6-10, 30/2.

   **NOT SUPPORTED:**

  * "#", "?", "W"
  * "L" (e.g. the last day of the month). Workaround is possible with multiple schedules, though beware of Feb 29.
  * symbolic definitions of weekday (Mon, Fri) and month (Jan, Sep) - only numeric is used.


* Year and seconds precision by default.

  If year is ommited, will try to find the next job run for every year from current to 10 years in the future.

  If second is ommited, by default it specifies 00.

* Location (Timezone) per schedule. Job can run in multiple timezones simultaneously.

  Note that we don't handle summer/winter daylight shifts, it is possible for job to skip its run or run twice during the shift. Recommended to use UTC.
  By default the location is Local, i.e. the one of the machine.

* Job control:

* Stdout and stderr redirection of running command to files. File rotation is not supported.
* Simultaneous run of job is disabled by default, change it with "parallel" key.
  * Job timeout after which the job will be killed.


  Based on uber/zapp library, with the ability to switch to structured (json) logging.
  Colored and not, internal information like function caller and line is available and also configurable.

* Instrumentation:

   * HTTP based management interface to pause, resume, or gracefully shutdown scheduler with configurable timeout to wait for jobs to finish.
   * Prometheus metrics.
   * Scheduler logging based on uber/zapp library, with the ability to switch to structured (json) logging.
   Logs rotation is not supported. Colored severity keywords, internal information like function caller and line are also configurable. Jobs run status and next scheduled time is reported.

## Installation

Fetch prebuilt binary from Releases page. Only Linux is avaliable for now.

Installation from the source:

```go get https://github.com/Tarick/tscheduler```

## Configuration

See [examples/config.yaml](https://github.com/Tarick/tscheduler/examples/config.yaml), should be self explanatory.

The path to file could passed via --config option or found in order in:
* current dir
* $HOME/.config/tscheduler/
* /etc/tscheduler/
