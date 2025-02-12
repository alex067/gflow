/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package config

import (
	"log/slog"
	"os"

	"github.com/alex067/gflow/internal/pkg/prompt"
	"github.com/spf13/cobra"
)

// getContextsCmd represents the getContexts command
var getContextsCmd = &cobra.Command{
	Use:   "get-contexts",
	Short: "Gets the current configured contexts set in the gflow file.",
	Run: func(cmd *cobra.Command, args []string) {
		err := configContext.ReadConfigFile(gcf)
		if err != nil {
			logger.Error(
				"Failed to read context file",
				slog.String("error", err.Error()),
			)
			os.Exit(1)
		}

		var mSelector prompt.MultiSelector
		mSelector.RunGetContextDisplay(configContext.CurrentContext, configContext.Contexts)
	},
}

func init() {
	ConfigCmd.AddCommand(getContextsCmd)
}
