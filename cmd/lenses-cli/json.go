package main

import (
	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func shouldPrintJSON(cmd *cobra.Command, fn func() (interface{}, error)) *cobra.Command {
	bite.CanPrintJSON(cmd)

	cmd.RunE = returnJSON(fn)
	return cmd
}

func returnJSON(fn func() (interface{}, error)) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		printValues, err := fn()
		if err != nil {
			return err
		}
		return bite.PrintJSON(cmd, printValues)
	}
}
