/*
Copyright © 2024 NAME HERE <EMAIL ADDRESS>
*/
package clear

import (
	"log/slog"
	"os"

	"github.com/alex067/gflow/internal/pkg/gcontext"
	"github.com/spf13/cobra"
)

var (
	gcf           gcontext.GConfigFile
	logger        *slog.Logger
	configContext gcontext.GConfigContext
)

// configCmd represents the config command
var ClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clears watcher resources on Grafana.",
}

func init() {
	logger = slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	gcf.Directory = ".gflow"
	gcf.Name = "config.yaml"

	ClearCmd.AddCommand(allCmd)
}
