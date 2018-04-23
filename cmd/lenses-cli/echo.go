package main

import (
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

var (
	// if true then commands will not output info messages, like "Processor ___ created".
	// Look the `echo` func for more, it's not a global flag but it's a common one, all commands that return info messages
	// set that via command flag binding.
	//
	// Defaults to false.
	silent bool
)

func canBeSilent(cmd *cobra.Command) *cobra.Command {
	cmd.Flags().BoolVar(&silent, "silent", false, "run in silent mode. No printing info messages for CRUD except errors, defaults to false")
	return cmd
}

func echo(cmd *cobra.Command, format string, args ...interface{}) error {
	if silent {
		return nil
	}

	if !strings.HasSuffix(format, "\n") {
		format += "\n" // add a new line.
	}

	_, err := fmt.Fprintf(cmd.OutOrStdout(), format, args...)
	return err
}
