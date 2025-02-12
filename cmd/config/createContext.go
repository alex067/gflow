/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package config

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alex067/gflow/internal/pkg/gcontext"
	"github.com/spf13/cobra"
)

// createContextCmd represents the createContext command
var createContextCmd = &cobra.Command{
	Use:   "create-context",
	Short: "Creates a new gflow context.",
	Run: func(cmd *cobra.Command, args []string) {
		absConfigPath, absConfigFilePath, err := gcf.GetAbsolutePath()
		if err != nil {
			logger.Error(err.Error())
			os.Exit(1)
		}

		// Search and/or create config diretory
		if _, err := os.Stat(absConfigPath); err != nil {
			if os.IsNotExist(err) {
				os.MkdirAll(absConfigPath, 0755)
				//isContextDirectoryExist = false
			} else {
				logger.Error("Failed searching for gflow config directory", slog.String("error", err.Error()))
				os.Exit(1)
			}
		}

		var newContext gcontext.GContext

		fmt.Print("Grafana Instance URL (Required): ")
		fmt.Scanln(&newContext.Url)
		fmt.Print("Context Name (Required): ")
		fmt.Scanln(&newContext.Name)
		fmt.Print("Dashboards Path (Required, Absolute): ")
		fmt.Scanln(&newContext.Context.Dashboards.Path)
		fmt.Print("Grafana Tenant (Required): ")
		fmt.Scanln(&newContext.Context.Dashboards.GrafanaTenant)
		fmt.Print("Grafana Auth Token (Required): ")
		fmt.Scanln(&newContext.Authentication.Grafana.Token)
		fmt.Print("Grafana Gflow Folder Uid (Optional): ")
		fmt.Scanln(&newContext.Context.Dashboards.GrafanResources.FolderUid)

		err = configContext.CreateNewContext(newContext, gcf)
		if err != nil {
			logger.Error("Failed to create new context", slog.String("error", err.Error()))
			os.Exit(1)
		}

		logger.Info("Created new context", newContext.Name, absConfigFilePath)

		// If single context set it as active
		contexts := configContext.GetContextNames()
		if len(contexts) == 1 {
			if err := configContext.SetCurrentContext(newContext.Name, false); err != nil {
				logger.Error("Failed to set current context", slog.String("error", err.Error()))
			}
		}
	},
}

func init() {
	ConfigCmd.AddCommand(createContextCmd)
}
