package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/landoop/lenses-go"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newLSQLCommand())
}

func newLSQLCommand() *cobra.Command {
	// Idea:
	// Maybe in the (near) future give a something like a --details flag in order
	// to print the whole information for the lsql validation(line,column and error message),
	// execution(data with offsets, the messages and so on)
	// and execution error (fromLine, toLine, fromColumn, toColumn and error message).
	// As far this is not requested, because the user expects to see the sql result's message
	// and an error as a text, but if will be requested, it can be done.
	var (
		validate bool
		// only on execution: if true then the LSQLStop will contain the offsets as well.
		withOffsets bool
		// only on execution: if not empty and > "1s" the client will accept LSQLStats every `statsEvery` duration, therefore they will be visible to the output.
		statsEvery time.Duration
	)

	rootSub := &cobra.Command{
		Use:           "sql [--validate?] [query]",
		Short:         "Execute or Validate Only Lenses query (LSQL) on the fly",
		Example:       `sql --offsets --stats=2s "SELECT * FROM reddit_posts LIMIT 50"`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			var query []byte

			// argument has a priority.
			if n := len(args); n == 1 {
				query, err = bite.TryReadFileContents(args[0])
				if err != nil {
					return err
				}
			} else if n == 0 {
				// read from input pipe, no argument given.
				has, b, err := bite.ReadInPipe()
				if err != nil {
					return fmt.Errorf("io pipe: %v", err)
				}

				if !has {
					// no data to read from.
					return fmt.Errorf("sql argument is missing and input pipe has no data to read from")
				}

				query = b
			} else {
				// argument and input pipe are missing.
				return fmt.Errorf("sql argument is the only one required argument")
			}

			if len(query) == 0 {
				return fmt.Errorf("query should not be empty")
			}

			// replace all new line with spaces and trim any trailing space.
			query = bytes.Replace(query, []byte("\n"), []byte(" "), -1)
			query = bytes.TrimSpace(query)

			// if --validate then validate, not execute.
			if validate {
				validation, err := client.ValidateLSQL(string(query))
				if err != nil {
					return err
				}

				if !validation.IsValid {
					// fmt.Fprintln(cmd.OutOrStdout(), validation.Message)
					// return it as error so Exit(1).
					return fmt.Errorf(validation.Message)
				}

				// print nothing if it was valid, remember:
				// smaller outputs is better, especially for external tools that
				// want to use the cli instead of the client api directly.
				return nil
			}

			recordHandler := func(r lenses.LSQLRecord) error {
				b := []byte(r.Value) // we care for the value here, which is a json raw string.
				var in interface{}
				if errR := json.Unmarshal(b, &in); errR != nil {
					return errR // fail on first error.
				}

				// return printJSON(cmd, in)
				return bite.PrintJSON(cmd, in) // if != nil then it will exit(1) and print the error. keep json?
				// or tableprinter.PrintJSON(cmd.OutOrStdout(), b) ?
			}

			stopHandler := func(stopRecord lenses.LSQLStop) error {
				/* Output (the "offsets" key is filled because ran with --offsets):
				Stop
				{
				  "isTimeRemaining": true,
				  "isTopicEnd": false,
				  "isStopped": false,
				  "totalRecords": 5,
				  "skippedRecords": 0,
				  "recordsLimit": 0,
				  "totalSizeRead": 2070,
				  "size": 2070,
				  "offsets": [
				    {
				      "partition": 2,
				      "min": 881405762,
				      "max": 910405850
				    },
				    {
				      "partition": 1,
				      "min": 860858539,
				      "max": 888810749
				    },
				    {
				      "partition": 0,
				      "min": 1212864063,
				      "max": 1242756366
				    }
				  ]
				}
				*/
				// here we stop but it's not an error, so we can't return a non-nil error.
				fmt.Fprintln(cmd.OutOrStdout(), "Stop")
				// printJSON(cmd, stopRecord)
				bite.PrintObject(cmd, stopRecord)
				return nil
			}

			stopErrHandler := func(errRecord lenses.LSQLError) error {
				fmt.Fprintln(cmd.OutOrStdout(), "Stop:Error")
				// this error will be catched by the err = client.LSQL(...) below, same with the rest of the handlers.
				return fmt.Errorf(errRecord.Message)
			}

			statsHandler := func(stats lenses.LSQLStats) error {
				/* Output (with --stats):
				Stats
				{
				  "totalRecords": 501,
				  "recordsSkipped": 0,
				  "recordsLimit": 0,
				  "totalBytes": 144875,
				  "maxSize": 9223372036854775807,
				  "currentSize": 144875
				}
				*/
				fmt.Fprintln(cmd.OutOrStdout(), "Stats")
				// return printJSON(cmd, stats)
				return bite.PrintObject(cmd, stats)
			}

			if statsEvery <= 0 {
				// disable stats (in-time) and stop record (at end, including its own stats).
				statsHandler = nil
				stopHandler = nil
			}

			return client.LSQL(string(query), withOffsets, statsEvery, recordHandler, stopHandler, stopErrHandler, statsHandler)
		},
	}

	rootSub.Flags().BoolVar(&validate, "validate", false, "runs query validation only") // if --validate exists in the flags then it's true.
	rootSub.Flags().BoolVar(&withOffsets, "offsets", false, "the stop output will contain the 'offsets' information as well")
	rootSub.Flags().DurationVar(&statsEvery, "stats", 0, "--stats=2s if passed the client will accept stats records every 'stats' duration, therefore they will be visible to the output")
	bite.CanPrintJSON(rootSub)

	rootSub.AddCommand(
		newGetRunningQueriesCommand(),
		newCancelQueryCommand(),
	)

	return rootSub
}

func newGetRunningQueriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "running",
		Short:         "Print the current running queries, if any",
		Example:       "sql running",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			queries, err := client.GetRunningQueries()
			if err != nil {
				return err
			}

			// return printJSON(cmd, queries)
			return bite.PrintObject(cmd, queries)
		},
	}

	bite.CanPrintJSON(cmd)
	return cmd
}

func newCancelQueryCommand() *cobra.Command {
	var id int64

	cmd := &cobra.Command{
		Use:           "cancel",
		Short:         "Cancels a running query by its ID. It returns true whether it was cancelled otherwise false or error",
		Example:       "sql cancel 42 or sql cancel --id=42",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == 0 {
				if len(args) < 1 {
					return fmt.Errorf("id is required")
				}
				var err error
				id, err = strconv.ParseInt(args[0], 10, 64)
				if err != nil {
					return err
				}
			}

			deleted, err := client.CancelQuery(id)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), deleted)
			return nil
		},
	}

	cmd.Flags().Int64Var(&id, "id", 0, "--id=42 cancel by id")

	return cmd
}
