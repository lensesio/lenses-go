// +build shell

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go"
	"github.com/spf13/cobra"
)

var sqlLiveStream, sqlStats, sqlKeys, sqlKeysOnly, sqlMeta bool
var gCmd *cobra.Command

type (
	responseWithKeysWithMeta struct {
		Key      json.RawMessage `json:"key"`
		Value    json.RawMessage `json:"value"`
		Metadata lenses.MetaData `json:"metadata"`
	}

	responseWithKeys struct {
		Key   json.RawMessage `json:"key"`
		Value json.RawMessage `json:"value"`
	}

	responseWithMeta struct {
		Value    json.RawMessage `json:"value"`
		Metadata lenses.MetaData `json:"metadata"`
	}

	responseWithKeysWithMetaOnly struct {
		Key      json.RawMessage `json:"key"`
		Metadata lenses.MetaData `json:"metadata"`
	}
)

func init() {
	app.AddCommand(newLiveLSQLCommand())
}

func readAndQuoteQueries(args []string) ([]string, error) {
	if n := len(args); n > 0 {
		queries := make([]string, n, n)
		for i, arg := range args {
			query, err := bite.TryReadFileContents(arg)
			if err != nil {
				return nil, err
			}
			// replace all new line with spaces and trim any trailing space.
			query = bytes.Replace(query, []byte("\n"), []byte(" "), -1)
			query = bytes.TrimSpace(query)
			queries[i] = string(query)
		}
		return queries, nil
	}

	// read from input pipe, no argument given.
	has, b, err := bite.ReadInPipe()
	if err != nil {
		return nil, fmt.Errorf("io pipe: [%v]", err)
	}

	if !has || len(b) == 0 {
		// no data to read from.
		return nil, fmt.Errorf("sql argument is missing and input pipe has no data to read from")
	}

	query := strconv.Quote(string(b))
	return []string{query}, nil
}

func runSQL(cmd *cobra.Command, sql string, meta bool, keys bool, keysOnly bool, liveStream bool, stats bool) error {
	currentConfig := configManager.config.GetCurrent()

	conn, err := lenses.OpenLiveConnection(lenses.LiveConfiguration{
		Token: client.Config.Token,
		Host:  currentConfig.Host,
		Debug: currentConfig.Debug,
		SQL:   sql,
		Live:  liveStream,
		Stats: 2,
	})

	if err != nil {
		return err
	}

	go func() {
		// print each error on screen, do not exit because
		// a query may be errorred but another, most important may running for a long time.
		select {
		case err := <-conn.Err():
			// ignore error and don't print that caused by ctrl/cmd+c while trying to read.
			if errNet, isNetworkClosed := err.(*net.OpError); isNetworkClosed && errNet.Op == "read" {
				if strings.Contains(errNet.Error(), "use of closed") {
					return
				}
			}

			fmt.Fprintf(cmd.OutOrStderr(), "[%s]\n", err)
		}
	}()

	// we exit on error, the only one place that we directly exit from here.
	errorReporter := func(resp lenses.LiveResponse) error {
		// parse it, otherwise it shows it very ugly.
		var errStr string
		json.Unmarshal(resp.Data.Value, &errStr)
		_, err = fmt.Fprintf(cmd.OutOrStderr(), "[%s]: [%s]\n", resp.Type, errStr)
		os.Exit(1)
		return err
	}

	// login error or anything? depends on the back-end.
	conn.OnError(errorReporter)
	conn.OnInvalidRequest(errorReporter)

	if stats {
		conn.OnStats(func(resp lenses.LiveResponse) error {
			err := bite.PrintJSON(cmd, resp)
			return err
		})
	}

	// first subscribe to any incoming kafka messages (as result of the lsql publish).
	conn.OnRecordMessage(func(resp lenses.LiveResponse) error {

		var data interface{}

		if keysOnly {
			// keys and metadata only
			if meta {
				data = responseWithKeysWithMetaOnly{
					Key:      resp.Data.Key,
					Metadata: resp.Data.Metadata,
				}
			} else {
				data = resp.Data.Key
			}
		} else {
			// data only
			if !keys && !meta {
				data = resp.Data.Value
			}

			// data and metadata
			if !keys && meta {
				data = responseWithMeta{
					Value:    resp.Data.Value,
					Metadata: resp.Data.Metadata,
				}
			}

			// keys and data
			if keys && !meta {
				data = responseWithKeys{
					Key:   resp.Data.Key,
					Value: resp.Data.Value,
				}
			}

			// keys, data and metadata
			if keys && meta {
				data = responseWithKeysWithMeta{
					Key:      resp.Data.Key,
					Value:    resp.Data.Value,
					Metadata: resp.Data.Metadata,
				}
			}
		}

		if err := bite.PrintJSON(cmd, data); err != nil {
			golog.Error(err)
			return err
		}

		return nil
	})

	conn.OnEnd(func(resp lenses.LiveResponse) error {
		if !interactiveShell && sqlLiveStream {
			os.Exit(0)
		} else {
			p, err := os.FindProcess(os.Getpid())
			if err != nil {
				return err
			}

			p.Signal(os.Interrupt)
		}
		conn.Close()
		return nil
	})

	ch := make(chan os.Signal, 1)
	signal.Notify(ch,
		// kill -SIGINT XXXX or Ctrl+c
		os.Interrupt,
		syscall.SIGINT, // register that too, it should be ok
		// os.Kill  is equivalent with the syscall.SIGKILL
		os.Kill,
		syscall.SIGKILL, // register that too, it should be ok
		// kill -SIGTERM XXXX
		syscall.SIGTERM,
	)

	return conn.Wait(ch)
}

func newLiveLSQLCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "query",
		Short:            "Queries, either browsing for continuous (live-stream)",
		Example:          `query "SELECT * FROM cc_payments LIMIT 10"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) < 1 {
				golog.Errorf(`sql query is missing, the correct form is: sql "your query"`)
				return nil
			}

			if len(args) > 1 {
				golog.Errorf(`Only one sql statement is allowed, received [%d]`, len(args))
				return nil
			}

			queries, err := readAndQuoteQueries(args)
			if err != nil {
				return err
			}

			if len(queries) == 0 {
				golog.Errorf("query should not be empty")
				return nil
			}

			// validate query
			validation, err := client.ValidateSQL(queries[0], 0)

			if err != nil {
				return err
			}

			checkValidation(validation)
			runSQL(cmd, queries[0], sqlMeta, sqlKeys, sqlKeysOnly, sqlLiveStream, sqlStats)
			return nil

		},
	}

	cmd.Flags().BoolVar(&sqlLiveStream, "live-stream", false, "Run in continuous query mode")
	cmd.Flags().BoolVar(&sqlStats, "stats", false, "Print query stats")
	cmd.Flags().BoolVar(&sqlKeys, "keys", false, "Print message keys")
	cmd.Flags().BoolVar(&sqlKeysOnly, "keys-only", false, "Print message keys only")
	cmd.Flags().BoolVar(&sqlMeta, "meta", false, "Print message metadata")

	bite.CanPrintJSON(cmd)

	return cmd
}
