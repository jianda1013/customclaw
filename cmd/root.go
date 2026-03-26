package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	configPath  string
	actionsPath string
)

var rootCmd = &cobra.Command{
	Use:   "customclaw",
	Short: "Local AI agent for automating workflows across external services",
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "config.json", "Path to config file")
	rootCmd.PersistentFlags().StringVar(&actionsPath, "actions", "actions.json", "Path to actions file")

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(chatCmd)
	rootCmd.AddCommand(validateCmd)
	rootCmd.AddCommand(toolsCmd)
}

func execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
