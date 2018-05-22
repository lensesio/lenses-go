package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

func init() {
	// TODO: start by adding this.
	rootCmd.AddCommand(newLiveLSQLCommand())
}

func readAndQuoteQueries(args []string) ([]string, error) {
	if n := len(args); n > 0 {
		queries := make([]string, n, n)
		for i, arg := range args {
			query, err := tryReadFileContents(arg)
			if err != nil {
				return nil, err
			}
			// replace all new line with spaces and trim any trailing space.
			query = bytes.Replace(query, []byte("\n"), []byte(" "), -1)
			query = bytes.TrimSpace(query)
			queries[i] = strconv.Quote(string(query))
		}
		return queries, nil
	}

	// read from input pipe, no argument given.
	has, b, err := readInPipe()
	if err != nil {
		return nil, fmt.Errorf("io pipe: %v", err)
	}

	if !has || len(b) == 0 {
		// no data to read from.
		return nil, fmt.Errorf("sql argument is missing and input pipe has no data to read from")
	}

	query := strconv.Quote(string(b))
	return []string{query}, nil
}

func newLiveLSQLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "live sql [query]",
		Short:            "Live sql provides \"real-time\" sql queries with your lenses box",
		Example:          exampleString(`live sql "SELECT * FROM cc_payments WHERE _vtype='AVRO' AND _ktype='STRING' AND _sample=2 AND _sampleWindow=200" "query2" "query3"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var queryArgs []string
			if len(args) <= 1 {
				// Detect if there are data coming from stdin:
				stats, _ := os.Stdin.Stat()
				if (stats.Mode() & os.ModeCharDevice) != os.ModeCharDevice {
					stdin, err := ioutil.ReadAll(os.Stdin)
					if err != nil {
						return fmt.Errorf(`failed to read from stdin and sql query is missing`)
					}

					rawArgs := strings.Split(string(stdin), ";")
					for _, v := range rawArgs {
						if len(strings.TrimSpace(v)) != 0 {
							queryArgs = append(queryArgs, v)
						}
					}

				} else {
					return fmt.Errorf(`sql query is missing, the correct form is: live sql "query here"`)
				}
			}

			if len(queryArgs) == 0 {
				queryArgs = args[1:] // -> omit the "sql" because it parsed as argument.
			}
			queries, err := readAndQuoteQueries(queryArgs)
			if err != nil {
				return err
			}

			if len(queries) == 0 {
				return fmt.Errorf("query should not be empty")
			}

			currentConfig := configManager.getCurrent()

			conn, err := lenses.OpenLiveConnection(lenses.LiveConfiguration{
				User:     currentConfig.User,
				Password: currentConfig.Password,
				Host:     currentConfig.Host,
				Debug:    currentConfig.Debug,
			})

			if err != nil {
				return err
			}

			go func() {
				// print each error on screen, do not exit because
				// a query may be errored but another, most important may running for a long time.
				select {
				case err := <-conn.Err():
					fmt.Fprintf(cmd.OutOrStderr(), "%s\n", err)
				}
			}()

			// we exit on error, the only one place that we directly exit from here.
			errorReporter := func(_ lenses.LivePublisher, resp lenses.LiveResponse) error {
				// parse it, otherwise it shows it very ungly.
				var errStr string
				json.Unmarshal(resp.Content, &errStr)
				_, err = fmt.Fprintf(cmd.OutOrStderr(), "%s: %s\n", resp.Type, errStr)
				os.Exit(1)
				return err
			}

			// login error or anything? depends on the back-end.
			conn.OnError(errorReporter)
			conn.OnInvalidRequest(errorReporter)

			// first subscribe to any incoming kafka messages (as result of the lsql publish).
			conn.OnKafkaMessage(func(_ lenses.LivePublisher, resp lenses.LiveResponse) error {
				b, err := resp.Content.MarshalJSON()
				if err != nil {
					return err
				}

				var data []lenses.LSQLRecord
				if err = json.Unmarshal(b, &data); err != nil {
					return err
				}

				for i := range data {
					b := []byte(data[i].Value)
					var in interface{}
					if err := json.Unmarshal(b, &in); err != nil {
						return err // fail on first error.
					}

					bb, err := json.MarshalIndent(in, "", "    ")
					if err != nil {
						return err // fail on first error.
					}
					result := string(bb)

					if _, err := fmt.Fprintf(cmd.OutOrStdout(), "%s\n", result); err != nil {
						return err
					}
				}

				return nil
			})

			// on login(`conn.Listen`) success send the lsql queries.
			conn.OnSuccess(func(pub lenses.LivePublisher, resp lenses.LiveResponse) error {
				if resp.CorrelationID == 2 {
					// if it comes as a result of the subscribe action,
					// print the topic(s) name.

					var name string
					if err := json.Unmarshal(resp.Content, &name); err != nil {
						return err
					}

					title := "Topic"
					if len(queries) > 1 {
						title += "s"
					}

					// ignore the topic names from the standard output
					// use the stderr for it:
					fmt.Fprintf(cmd.OutOrStderr(), "%s: %s\n", title, name)
					return nil
				}

				// we can use it to return results from many lsqueries,
				// it works, it returns results but it's not recommended, cpu goes really high!
				// lenses-cli live sql
				// "SELECT * FROM cc_payments WHERE _vtype='AVRO' AND _ktype='STRING' AND _sample=2 AND _sampleWindow=200"
				// "SELECT * FROM reddit_posts WHERE _vtype='AVRO' AND _ktype='AVRO' AND _sample=2 AND _sampleWindow=200"
				content := fmt.Sprintf(`{"sqls": [%s]}`, strings.Join(queries, ","))
				return pub.Publish(lenses.SubscribeRequest, 2, content)
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
		},
	}

	return cmd
}
