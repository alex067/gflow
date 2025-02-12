/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package clear

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/alex067/gflow/internal/pkg/gclient"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
)

var (
	gContext string
	gc       *gclient.GrafanaClient
)

var allCmd = &cobra.Command{
	Use:   "all",
	Short: "Clears all watcher resources on Grafana.",
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
		}

		currentContextConfig, err := configContext.GetContext(gContext)
		if err != nil {
			logger.Error("Failed to read current context", slog.String("error", err.Error()))
			os.Exit(1)
		}

		gc = &gclient.GrafanaClient{
			Url:      "https://grafana.o11y.web3factory.consensys.net",
			TenantId: currentContextConfig.Context.Dashboards.GrafanaTenant,
			ApiKey:   currentContextConfig.Authentication.Grafana.Token,
			Logger:   logger,
			HttpClient: &http.Client{
				Timeout: 60 * time.Second,
			},
		}
	},
	Run: func(cmd *cobra.Command, args []string) {
		watcherDashboards := configContext.GetWatchedDashboards()

		if len(watcherDashboards) > 0 {
			// Check for valid dashboard UID in json file
			logger.Info(fmt.Sprintf("Clearing %d watcher dashboards from Grafana", len(watcherDashboards)))
			eg := errgroup.Group{}
			for _, val := range watcherDashboards {
				eg.Go(func() error {
					dbClient := &gclient.GrafanaDashboardClient{}
					dbClient.Uid = val.Uid
					if err := gc.DeleteWatcherDashboard(dbClient); err != nil {
						return fmt.Errorf("dashboard delete error, uid=%s, error=%v", val.Uid, err)
					}
					return nil
				})
				eg.Go(func() error {
					if err := configContext.ClearResourceDashboardByPath(val.Path); err != nil {
						return fmt.Errorf(
							"clear dashboard config error, uid=%s, error =%v",
							val.Uid,
							err,
						)
					}
					return nil
				})
			}

			if err := eg.Wait(); err != nil {
				logger.Error(err.Error())
			} else {
				logger.Info("Successfully removed watcher dashboards from Grafana")
			}
		} else {
			logger.Info("No watcher dashboards, aborting operation")
		}
	},
}

func init() {
	allCmd.Flags().StringVarP(&gContext, "context", "c", "", "Override current context")
}
