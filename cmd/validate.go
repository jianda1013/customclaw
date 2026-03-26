package main

import (
	"customclaw/internal/config"
	"fmt"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate config.json and actions.json",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("config: %w", err)
		}
		if err := cfg.Validate(); err != nil {
			return fmt.Errorf("config: %w", err)
		}
		fmt.Println("config.json OK")

		actions, err := config.LoadActions(actionsPath)
		if err != nil {
			return fmt.Errorf("actions: %w", err)
		}
		fmt.Printf("actions.json OK — %d tool(s), %d workflow(s)\n", len(actions.Tools), len(actions.Workflows))
		return nil
	},
}
