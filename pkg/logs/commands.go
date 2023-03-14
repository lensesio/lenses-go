package logs

import (
	"github.com/lensesio/bite"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewLogsCommandGroup creates `logs` command
func NewLogsCommandGroup() *cobra.Command {

	root := &cobra.Command{
		Use:              "logs",
		Short:            "List the info or metrics logs",
		Example:          `logs info`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	asObjects := root.PersistentFlags().Bool("no-text", false, "no-text will print as objects, defaults to false")
	logsInfoSubComamnd := NewGetLogsInfoCommand(asObjects)
	// if not `info $subCommand` passed then by-default show the info logs.
	root.RunE = logsInfoSubComamnd.RunE

	root.AddCommand(logsInfoSubComamnd)
	root.AddCommand(NewGetLogsMetricsCommand(asObjects))
	return root
}

// NewGetLogsInfoCommand creates `logs info` command
func NewGetLogsInfoCommand(asObjects *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "info",
		Short:            "List the latest (512) INFO logs",
		Example:          `logs info`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logs, err := config.Client.GetLogsInfo()
			if err != nil {
				return err
			}

			if *asObjects {
				return bite.PrintObject(cmd, logs)
			}

			return utils.PrintLogLines(logs)
		},
	}

	return cmd
}

// NewGetLogsMetricsCommand creates `logs metrics` command
func NewGetLogsMetricsCommand(asObjects *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "metrics",
		Short:            "List the latest (512) METRICS logs",
		Example:          `logs metrics`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logs, err := config.Client.GetLogsMetrics()
			if err != nil {
				return err
			}

			if *asObjects {
				return bite.PrintObject(cmd, logs)
			}

			return utils.PrintLogLines(logs)
		},
	}

	return cmd
}
