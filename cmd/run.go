package main

import (
	"customclaw/internal/triggers"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var runCmd = &cobra.Command{
	Use:   "run <command>",
	Short: "Run a one-shot command",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		_, actions, ag, err := bootstrap(configPath, actionsPath)
		if err != nil {
			return err
		}
		cli := triggers.NewCLITrigger(actions, ag)
		command := strings.Join(args, " ")
		fmt.Printf("running: %s\n\n", command)
		return cli.Run(command)
	},
}
