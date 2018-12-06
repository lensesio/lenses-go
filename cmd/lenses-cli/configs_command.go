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
		Aliases:          []string{"config"},
		Short:            "Print the whole lenses box configs",
		Example:          "configs",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				configEntryName := args[0]
				if configEntryName == commandModeName {
					// means that something like `config mode` called,
					// let's support it here as well, although
					// mode has its own command `mode` because it's super important
					// and users should call that instead.
					return newGetModeCommand().Execute()
				}

				var value interface{}
				err := client.GetConfigEntry(&value, configEntryName)
				if err != nil {
					return fmt.Errorf("retrieve config value [%s] failed: [%v]", configEntryName, err)
				}

				return bite.PrintJSON(cmd, value) // keep json.
			}

			config, err := client.GetConfig()
			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, config)
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
