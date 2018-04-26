package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jmespath/go-jmespath"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// if true then it doesn't prints json result(s) with indent.
	// Defaults to false.
	// It's not a global flag, but it's a common one, all commands that return results
	// use that via command flag binding.
	noPretty bool
	// jmespathQuery query to further filter any results, if any.
	// It's not a global flag, but it's a common one, all commands that return results
	// set that via command flag binding.
	jmespathQuery string

	jsonFlagSet = newFlagGroup("flagset.json", func(flags *pflag.FlagSet) {
		flags.BoolVar(&noPretty, "no-pretty", noPretty, "disable the pretty format for JSON output of commands (default false).")
		flags.StringVarP(&jmespathQuery, "query", "q", "", "a jmespath query expression. This allows for querying the JSON output of commands")
	})
)

func newFlagGroup(name string, register func(flags *pflag.FlagSet)) *pflag.FlagSet {
	flags := pflag.NewFlagSet(name, pflag.ExitOnError)
	register(flags)
	return flags
}

func canPrintJSON(cmd *cobra.Command) {
	cmd.Flags().AddFlagSet(jsonFlagSet)
}

func shouldPrintJSON(cmd *cobra.Command, fn func() (interface{}, error)) *cobra.Command {
	canPrintJSON(cmd)

	cmd.RunE = returnJSON(fn)
	return cmd
}

func returnJSON(fn func() (interface{}, error)) func(cmd *cobra.Command, args []string) error {
	return func(cmd *cobra.Command, args []string) error {
		printValues, err := fn()
		if err != nil {
			return err
		}
		return printJSON(cmd, printValues)
	}
}

func printJSON(cmd *cobra.Command, v interface{}) error {
	rawJSON, err := toJSON(v, !noPretty, jmesQuery(jmespathQuery, v))
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(cmd.OutOrStdout(), string(rawJSON))
	return err
}

type transformer func([]byte, bool) ([]byte, error)

func toJSON(v interface{}, pretty bool, transformers ...transformer) ([]byte, error) {
	var (
		rawJSON []byte
		err     error
	)

	if pretty {
		rawJSON, err = json.MarshalIndent(v, "", "  ")
		if err != nil {
			return nil, err
		}
	} else {
		rawJSON, err = json.Marshal(v)
		if err != nil {
			return nil, err
		}
	}

	for _, transformer := range transformers {
		if transformer == nil {
			continue // may give a nil transformer in variadic input.
		}
		b, err := transformer(rawJSON, pretty)
		if err != nil {
			return nil, err
		}
		if len(b) == 0 {
			continue
		}
		rawJSON = b
	}

	return rawJSON, err
}

func jmesQuery(query string, v interface{}) transformer {
	return func(rawJSON []byte, pretty bool) ([]byte, error) {
		if query == "" || strings.TrimSpace(string(rawJSON)) == "[]" { // if it's empty, exit.
			return nil, nil // don't throw error here, just skip it by returning nil result and nil error.
		}

		result, err := jmespath.Search(query, v)
		if err != nil {
			return nil, err
		}

		return toJSON(result, pretty)
	}
}
