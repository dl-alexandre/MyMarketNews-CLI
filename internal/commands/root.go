package commands

import (
	"fmt"
	"os"
	"time"

	"mpr/internal/config"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mpr",
	Short: "CLI for USDA AMS MPR Datamart",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("base-url", config.DefaultBaseURL, "API base URL")
	rootCmd.PersistentFlags().Duration("timeout", 30*time.Second, "HTTP request timeout")
	rootCmd.PersistentFlags().Float64("rps", 1.0, "Requests per second limit")
	rootCmd.PersistentFlags().Duration("cache-ttl", 12*time.Hour, "Reports cache TTL")
	rootCmd.PersistentFlags().String("cache-dir", "", "Cache directory (defaults to OS user cache dir)")

	rootCmd.AddCommand(reportsCmd())
	rootCmd.AddCommand(urlCmd())
	rootCmd.AddCommand(curlCmd())
	rootCmd.AddCommand(getCmd())
	rootCmd.AddCommand(correctionsCmd())
	rootCmd.AddCommand(recentCmd())
	rootCmd.AddCommand(emailCmd())
	rootCmd.AddCommand(timeseriesCmd())
	rootCmd.AddCommand(completionCmd())
}
