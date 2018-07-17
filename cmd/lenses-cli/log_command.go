package main

import (
	"fmt"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newLogsCommandGroup())
}

func newLogsCommandGroup() *cobra.Command {

	root := &cobra.Command{
		Use:              "logs",
		Short:            "List the info or metrics logs",
		Example:          `logs info`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	asObjects := root.PersistentFlags().Bool("no-text", false, "no-text will print as objects (json if --machine-friendly or table otherwise), defaults to false")

	var (
		logsInfoSubComamnd    = newGetLogsInfoCommand(asObjects)
		logsMetricsSubCommand = newGetLogsMetricsCommand(asObjects)
	)

	// if not `info $subCommand` passed then by-default show the info logs.
	root.RunE = logsInfoSubComamnd.RunE

	root.AddCommand(logsInfoSubComamnd)
	root.AddCommand(logsMetricsSubCommand)
	return root
}

func richLog(level string, log string) {
	switch strings.ToLower(level) {
	case "info":
		golog.Infof(log)
	case "warn":
		golog.Warnf(log)
	case "error":
		golog.Errorf(log)
	default:
		app.Print(log)
	}
}

func printLogLines(logs []lenses.LogLine) error {
	golog.SetTimeFormat("")

	for _, logLine := range logs {
		line := fmt.Sprintf("%s %s", logLine.Time, logLine.Message)
		richLog(logLine.Level, line)
	}

	return nil
}

func newGetLogsInfoCommand(asObjects *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "info",
		Short:            "List the latest (512) INFO logs",
		Example:          `logs info`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logs, err := client.GetLogsInfo()
			if err != nil {
				return err
			}

			if *asObjects {
				return bite.PrintObject(cmd, logs)
			}

			return printLogLines(logs)
		},
	}

	return cmd
}

func newGetLogsMetricsCommand(asObjects *bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "metrics",
		Short:            "List the latest (512) METRICS logs",
		Example:          `logs metrics`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			logs, err := client.GetLogsMetrics()
			if err != nil {
				return err
			}

			if *asObjects {
				return bite.PrintObject(cmd, logs)
			}

			return printLogLines(logs)
		},
	}

	return cmd
}
