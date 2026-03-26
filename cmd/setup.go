package main

import (
	"customclaw/internal/setup"
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive configuration wizard",
	Long:  "Walk through an interactive setup to configure your LLM provider, API keys, and integrations. Writes config.json.",
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := os.Stat(configPath); err == nil {
			overwrite, _ := cmd.Flags().GetBool("overwrite")
			if !overwrite {
				fmt.Printf("%s already exists. Run with --overwrite to reconfigure.\n", configPath)
				return nil
			}
		}
		w, err := setup.NewWizard()
		if err != nil {
			return err
		}
		defer w.Close()
		return w.Run(configPath)
	},
}

func init() {
	setupCmd.Flags().Bool("overwrite", false, "Overwrite existing config.json")
}
