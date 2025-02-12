/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alex067/gflow/internal/pkg/prompt"
	"github.com/spf13/cobra"
)

var useContextCmd = &cobra.Command{
	Use:   "use-context",
	Short: "Set the current-context in the gflow file.",
	Run: func(cmd *cobra.Command, args []string) {
		err := configContext.ReadConfigFile(gcf)
		if err != nil {
			logger.Error("Failed to read context file", slog.String("error", err.Error()))
			os.Exit(1)
		}

		var selectedContext string

		if cmd.Flag("context").Value.String() == "" {
			var mSelector prompt.MultiSelector
			selectedContext, err = mSelector.RunContextSelectMenu(configContext.CurrentContext, configContext.Contexts)
			if err != nil {
				logger.Error("Error processing context selector", slog.String("error", err.Error()))
				os.Exit(1)
			}
		} else {
			_, err := configContext.SearchContext(cmd.Flag("context").Value.String())
			if err != nil {
				logger.Error("Failed to find provided context in user config file")
				os.Exit(1)
			}
			selectedContext = cmd.Flag("context").Value.String()
			fmt.Println("✔ Selected context:", selectedContext)
		}
		configContext.SetCurrentContext(selectedContext, false)
	},
}

func init() {
	ConfigCmd.AddCommand(useContextCmd)
	useContextCmd.Flags().StringP("context", "c", "", "Set the context name")
}
