package cmd

import (
	"fmt"
	"os"

	"github.com/Tarick/tscheduler/job"

	homedir "github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	cfgFile    string
	verbose    bool
	jobConfigs []jobConfig
	metrics    metricsConfig
	management managementConfig
	log        *zap.SugaredLogger
)

type jobConfig struct {
	ID           string             `mapstructure:"id"`
	Command      []string           `mapstructure:"command"`
	Stdout       string             `mapstructure:"stdout"`
	Stderr       string             `mapstructure:"stderr"`
	Parallel     bool               `mapstructure:"parallel"`
	Timeout      string             `mapstructure:"timeout"`
	ScheduleSpec []job.ScheduleSpec `mapstructure:"schedule"`
}
type metricsConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Address string `mapstructure:"address"`
}
type managementConfig struct {
	Enabled                bool   `mapstructure:"enabled"`
	Address                string `mapstructure:"address"`
	SchedulerStopTimeout   uint16 `mapstructure:"scheduler_stop_timeout"`
	JobsTerminationTimeout uint16 `mapstructure:"jobs_termination_timeout"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "tscheduler",
	Short:   "Flexible tasks/jobs scheduler",
	Long:    `Command line jobs scheduler with per year and seconds precision, multiple schedules and commands control`,
	Version: "latest",
	// Uncomment the following line if your bare application
	// has an action associated with it:
	//	Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
func init() {
	cobra.OnInitialize(initConfig)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.config/tscheduler/config.yaml)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := homedir.Dir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		viper.AddConfigPath(".")      // optionally look for config in the working directory
		viper.SetConfigName("config") // name of config file (without extension)
		// Search config in home directory with name ".config/tscheduler/config.yaml"
		viper.AddConfigPath(home + ".config/tscheduler/")
		// viper.SetConfigType("yaml") // REQUIRED if the config file does not have the extension in the name
		viper.AddConfigPath("/etc/tscheduler/") // path to look for the config file in
	}
	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("FATAL: error in config file %s. %s", viper.ConfigFileUsed(), err)
		os.Exit(1)
	}

	fmt.Println("Using config file:", viper.ConfigFileUsed())

	if viper.IsSet("jobs") {
		viper.UnmarshalKey("jobs", &jobConfigs)
	} else {
		fmt.Println("Fatal: jobs are not defined in config file")
		os.Exit(1)
	}
	if viper.IsSet("metrics") {
		viper.UnmarshalKey("metrics", &metrics)
	} else {
		metrics.Enabled = false
	}
	if viper.IsSet("management") {
		viper.UnmarshalKey("management", &management)
	} else {
		management.Enabled = false
	}
	log = initLogger()
}

func initLogger() *zap.SugaredLogger {
	type LogConfig struct {
		Development       bool     `mapstructure:"development"`
		Level             string   `mapstructure:"level"`
		Encoding          string   `mapstructure:"encoding"`
		DisableCaller     bool     `mapstructure:"disable_caller"`
		DisableStacktrace bool     `mapstructure:"disable_stacktrace"`
		DisableColor      bool     `mapstructure:"disable_color"`
		OutputPaths       []string `mapstructure:"output_paths"`
		ErrorOutputPaths  []string `mapstructure:"error_output_paths"`
	}
	logcfg := LogConfig{}
	if viper.IsSet("logging") {
		err := viper.UnmarshalKey("logging", &logcfg)
		if err != nil {
			fmt.Println("Failure reading 'logging' configuration:", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("'logging:' configuration is missing in config file")
		os.Exit(1)
	}
	zapCfg := zap.Config{}
	zapCfg.Encoding = logcfg.Encoding
	zapCfg.Development = logcfg.Development
	zapCfg.DisableCaller = logcfg.DisableCaller
	zapCfg.DisableStacktrace = logcfg.DisableStacktrace
	var zapLvl zapcore.Level
	if err := zapLvl.UnmarshalText([]byte(logcfg.Level)); err != nil {
		fmt.Println("Incorrect logging.level value,", logcfg.Level)
		os.Exit(1)
	}
	zapCfg.Level = zap.NewAtomicLevelAt(zapLvl)
	zapCfg.OutputPaths = logcfg.OutputPaths
	zapCfg.ErrorOutputPaths = logcfg.ErrorOutputPaths
	zapCfg.EncoderConfig = zapcore.EncoderConfig{}
	zapCfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	zapCfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	if logcfg.DisableColor || logcfg.Encoding == "json" {
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	} else {
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}
	zapCfg.EncoderConfig.TimeKey = "time"
	zapCfg.EncoderConfig.MessageKey = "message"
	zapCfg.EncoderConfig.LevelKey = "severity"
	zapCfg.EncoderConfig.CallerKey = "caller"
	// fmt.Printf("%+v\n", zapCfg)
	logger, err := zapCfg.Build()
	if err != nil {
		fmt.Println("Failure initialising logger:", err)
		os.Exit(1)
	}
	defer logger.Sync()
	return logger.Sugar()
}
