package main

import (
	"fmt"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newGetConfigsCommand())
	app.AddCommand(newGetModeCommand())
}

func newGetConfigsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "configs",
		Short:            "Print the whole lenses box configs",
		Example:          "configs",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				if mode := args[0]; mode == commandModeName {
					// means that something like `config mode` called,
					// let's support it here as well, although
					// mode has its own command `mode` because it's super important
					// and users should call that instead.
					return newGetModeCommand().Execute()
				}

				var value interface{}
				if err := client.GetConfigEntry(&value, args[0]); err == nil {
					return bite.PrintJSON(cmd, value) // keep json?
					// if error or no valid key then continue with printing the whole lenses configuration.
				}

			}

			config, err := client.GetConfig()
			if err != nil {
				return err
			}

			// print all as json, it's not so much a visual-required command.
			return bite.PrintJSON(cmd, config)
		},
	}

	bite.CanPrintJSON(cmd)

	return cmd
}

const commandModeName = "mode"

func newGetModeCommand() *cobra.Command {
	return &cobra.Command{
		Use:                   commandModeName,
		Short:                 "Print the configuration's execution mode",
		Example:               commandModeName,
		DisableFlagParsing:    true,
		DisableFlagsInUseLine: true,
		DisableSuggestions:    true,
		TraverseChildren:      false,
		RunE: func(cmd *cobra.Command, args []string) error {
			mode, err := client.GetExecutionMode()
			if err != nil {
				return err
			}
			_, err = fmt.Fprintln(cmd.OutOrStdout(), string(mode))
			return err
		},
	}
}
