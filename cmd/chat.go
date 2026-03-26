package main

import (
	"customclaw/internal/triggers"

	"github.com/spf13/cobra"
)

var chatCmd = &cobra.Command{
	Use:   "chat",
	Short: "Start an interactive REPL session",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, actions, ag, err := bootstrap(configPath, actionsPath)
		if err != nil {
			return err
		}
		cli := triggers.NewCLITrigger(actions, ag)
		return cli.Chat()
	},
}
