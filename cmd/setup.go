package main

import (
	"customclaw/internal/setup"

	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive configuration wizard",
	Long:  "Configure your LLM provider, API keys, and integrations. If config.json already exists its current values are shown as defaults — press Enter to keep any field unchanged.",
	RunE: func(cmd *cobra.Command, args []string) error {
		w, err := setup.NewWizard()
		if err != nil {
			return err
		}
		defer w.Close()
		return w.Run(configPath)
	},
}
