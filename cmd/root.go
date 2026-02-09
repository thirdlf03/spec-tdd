package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thirdlf03/spec-tdd/internal/logger"
)

var (
	cfgFile   string
	debug     bool
	logFormat string
	appLogger *logger.Logger
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "spec-tdd",
	Short: "Spec-driven TDD helper",
	Long: `Spec-driven TDD helper that keeps requirements, examples, and test scaffolds in sync.

It generates test skeletons from a YAML DSL and produces traceability reports
to catch missing coverage early.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig, initLogger)

	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./config/config.yaml)")
	rootCmd.PersistentFlags().BoolVar(&debug, "debug", false, "enable debug mode")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "text", "log format (text or json)")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in current directory and ./config with name "config" (without extension).
		viper.AddConfigPath(".")
		viper.AddConfigPath("./config")
		viper.AddConfigPath(home)
		viper.SetConfigType("yaml")
		viper.SetConfigName("config")
	}

	// Read in environment variables that match
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}
}

// initLogger initializes the global logger
func initLogger() {
	appLogger = logger.NewFromFlags(debug, logFormat)
	slog.SetDefault(appLogger.Logger)
}

// GetLogger returns the global logger
func GetLogger() *logger.Logger {
	if appLogger == nil {
		appLogger = logger.NewFromFlags(false, "text")
	}
	return appLogger
}
