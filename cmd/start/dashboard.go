/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package start

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"github.com/alex067/gflow/internal/pkg/gclient"
	"github.com/alex067/gflow/internal/pkg/prompt"
	"github.com/spf13/cobra"
)

var (
	dashboardFile string
	gContext      string
	gc            *gclient.GrafanaClient
)

type GrafanaDashboardJson struct {
	Uid     string `json:"uid"`
	Version int    `json:"version"`
}

// startCmd represents the start command
var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Watch and sync dashboard changes.",
	PreRun: func(cmd *cobra.Command, args []string) {
		err := configContext.ReadConfigFile(gcf)
		if err != nil {
			logger.Error("Failed to read config file", slog.String("error", err.Error()))
			os.Exit(1)
		}

		if cmd.Flag("context").Value.String() == "" {
			gContext = configContext.CurrentContext
			if gContext == "" {
				logger.Error("Run config use-context to set the current context or supply the context to use")
				os.Exit(1)
			}
		} else {
			configContext.SetCurrentContext(gContext, true)
		}

		interval, err := strconv.Atoi(cmd.Flag("interval").Value.String())
		if err != nil {
			fmt.Fprintf(os.Stderr, "invalid interval value: %v", err)
			os.Exit(1)
		}

		currentContextConfig, err := configContext.GetContext(gContext)
		if err != nil {
			logger.Error("Failed to read current context", slog.String("error", err.Error()))
			os.Exit(1)
		}

		gc = &gclient.GrafanaClient{
			Url:      currentContextConfig.Url,
			TenantId: currentContextConfig.Context.Dashboards.GrafanaTenant,
			ApiKey:   currentContextConfig.Authentication.Grafana.Token,
			Interval: time.Duration(interval) * time.Second,
			Logger:   logger,
			HttpClient: &http.Client{
				Timeout: 60 * time.Second,
			},
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		logger.Info("Starting dashboard watcher process")

		currentContextConfig, err := configContext.GetContext(gContext)
		if err != nil {
			logger.Error("Failed to read current context", slog.String("error", err.Error()))
			os.Exit(1)
		}

		var dashboardFilePath string

		if dashboardFile == "" {
			// Display multi select menu
			var mSelector prompt.MultiSelector
			dashboardFilePath, err = mSelector.RunDashboardSelectMenu(
				currentContextConfig.Context.Dashboards.Path,
				configContext.GetWatchedDashboards(),
			)
			if err != nil {
				logger.Error("Failed to select dashboard", slog.String("error", err.Error()))
				os.Exit(1)
			}
		} else {
			// Search for dashboard based on filename
			dashboardFilePath = filepath.Join(currentContextConfig.Context.Dashboards.Path, dashboardFile)
			_, err := os.ReadFile(dashboardFilePath)
			if err != nil {
				logger.Error(
					"Failed to read dashboard file",
					slog.String("path", dashboardFilePath),
					slog.String("error", err.Error()),
				)
				os.Exit(1)
			}
		}

		var grafanaDashboard GrafanaDashboardJson
		// Ignore error since file is validated
		dashboardFileData, _ := os.ReadFile(dashboardFilePath)
		if err := json.Unmarshal(dashboardFileData, &grafanaDashboard); err != nil {
			logger.Error("Failed to parse dashboard file", slog.String("error", err.Error()))
			os.Exit(1)
		}

		if grafanaDashboard.Uid == "" {
			logger.Error(
				"Dashboard uid attribute not found in given config file",
				slog.String("dashboard", dashboardFilePath),
			)
			os.Exit(1)
		}

		// Begin watch process
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		done := make(chan error, 1)
		logger.Info("Starting watcher process")
		logger.Info("Interrupt the process to save current changes to local dashboard config file")

		dbClient := &gclient.GrafanaDashboardClient{}
		dbClient.FilePath = dashboardFilePath
		dbClient.FolderUid = currentContextConfig.Context.Dashboards.GrafanResources.FolderUid

		go func() {
			done <- gc.StartWatchingDashboard(ctx, configContext, dbClient)
		}()

		exitErr := <-done
		if exitErr != nil {
			if exitErr == context.Canceled || exitErr == gclient.ErrCleanShutdown {
				logger.Info("Saving final changes to disk")
				gc.GetDashboardChanges(dbClient)
				gc.SaveChangesToDisk(dbClient)
				// Clear state from config and remove dashboard from Grafana
				var wg sync.WaitGroup
				wg.Add(2)
				go func() {
					defer wg.Done()
					err := configContext.ClearResourceDashboardByPath(dbClient.FilePath)
					if err != nil {
						logger.Error(
							"failed clearing resource from config",
							slog.String("error", err.Error()))
					}
				}()
				go func() {
					defer wg.Done()
					err := gc.DeleteWatcherDashboard(dbClient)
					if err != nil {
						logger.Error(
							"failed  deleting dashboard",
							slog.String("error", err.Error()))
					}
				}()

				wg.Wait()
			} else {
				fmt.Fprintf(os.Stderr, "shutting down process: %v", exitErr)
			}
		}
	},
}

func init() {
	dashboardCmd.Flags().Int("interval", 10, "Grafana polling interval")
	dashboardCmd.Flags().StringVarP(&gContext, "context", "c", "", "Override current context")
	dashboardCmd.Flags().StringVarP(&dashboardFile, "dashboard", "d", "", "Grafana dashboard file relative path to watch (ex: example/foobar.json)")
}
