package cmd

import (
	"os"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/Infinite-Locus-Product/coca_cola_backend/config"
	"github.com/Infinite-Locus-Product/coca_cola_backend/vendors"
)

var rootCmd = &cobra.Command{
	Use:   "app",
	Short: "CLI for running and managing the application",
}

var ENVIRONMENT = config.GetConfig().AppEnv

func init() {
	cobra.OnInitialize(initLogging)
	cobra.OnInitialize(initVendors) // Placeholder for vendor initialization
}

// initLogging sets up the logging configuration, including Sentry integration
func initLogging() {
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetReportCaller(true)
	logrus.SetOutput(os.Stdout)

	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
		ForceColors:     true,
	})
}

// initVendors is a placeholder function for initializing third-party vendors
func initVendors() {
	vendors.Setup()
}

// Execute runs the root command
func Execute() error {
	return rootCmd.Execute()
}
