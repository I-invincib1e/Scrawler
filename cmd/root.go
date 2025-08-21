package cmd

import (
	"os"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "scraper",
	Short: "A web scraper CLI tool",
	Long: `A powerful web scraper with crawling and extraction capabilities.
Supports concurrent crawling, robots.txt testing, and multiple output formats.`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		color.Red("✘ Error: %s", err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here if needed
}
