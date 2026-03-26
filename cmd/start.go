package main

import (
	"customclaw/internal/triggers"
	"log"

	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the webhook server",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, actions, ag, err := bootstrap(configPath, actionsPath)
		if err != nil {
			return err
		}
		server := triggers.NewWebhookServer(cfg, actions, ag)
		log.Printf("using LLM provider: %s (%s)", cfg.LLM.Provider, cfg.LLM.Model)
		return server.Start()
	},
}
