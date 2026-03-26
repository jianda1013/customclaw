package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "List all available tools",
	RunE: func(cmd *cobra.Command, args []string) error {
		registry, err := buildRegistry(configPath)
		if err != nil {
			return err
		}
		fmt.Printf("%-28s %s\n", "NAME", "DESCRIPTION")
		fmt.Printf("%-28s %s\n", "----", "-----------")
		for _, t := range registry.All() {
			fmt.Printf("%-28s %s\n", t.Name(), t.Description())
		}
		return nil
	},
}
