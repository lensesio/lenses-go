package sql

import (
	"fmt"
	"os"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	"github.com/spf13/cobra"
)

var (
	version  string
	revision string
)

// LivePrefixState is a container for the interactive prompt state
var LivePrefixState struct {
	LivePrefix string
	IsEnable   bool
}

var sqlQuery string

//Executor sturct
type Executor struct {
	interactiveCmd *cobra.Command
	client         *api.Client
	sqlHistoryPath string
}

//NewExecutor creates a new executor
func NewExecutor(interactiveCmd *cobra.Command, client *api.Client, sqlHistoryPath string) *Executor {
	return &Executor{
		interactiveCmd: interactiveCmd,
		client:         client,
		sqlHistoryPath: sqlHistoryPath,
	}
}

//ChangeLivePrefix changes the prefix
func (e *Executor) ChangeLivePrefix() (string, bool) {
	return LivePrefixState.LivePrefix, LivePrefixState.IsEnable
}

//Execute execute an SQL query
func (e *Executor) Execute(sql string) {
	if strings.HasPrefix(sql, "!") {
		trimmed := strings.Trim(sql, " ")

		if trimmed == "!options" {
			fmt.Printf("Options: keys=%t, keysOnly=%t, meta=%t, stats=%t, live-stream=%t\n", sqlKeys, sqlKeysOnly, sqlMeta, sqlStats, sqlLiveStream)
			return
		}

		if trimmed == "!pretty" {

			if bite.GetJSONPrettyFlag(e.interactiveCmd) {
				e.interactiveCmd.Flags().Set("pretty", "false")
			} else {
				e.interactiveCmd.Flags().Set("pretty", "true")
			}

			fmt.Printf("Option [%s] set to [%t]\n", trimmed, bite.GetJSONPrettyFlag(e.interactiveCmd))
			return
		}

		if trimmed == "!keys" {
			if sqlKeys {
				sqlKeys = false
			} else {
				sqlKeys = true
			}

			fmt.Printf("Option [%s] set to [%t]\n", trimmed, sqlKeys)
			return
		}

		if trimmed == "!keys-only" {
			if sqlKeysOnly {
				sqlKeysOnly = false
			} else {
				sqlKeysOnly = true
			}

			fmt.Printf("Option [%s] set to [%t]\n", trimmed, sqlKeysOnly)
			return
		}

		if trimmed == "!meta" {

			if sqlMeta {
				sqlMeta = false
			} else {
				sqlMeta = true
			}

			fmt.Printf("Option [%s] set to [%t]\n", trimmed, sqlMeta)
			return
		}

		if trimmed == "!stats" {
			if sqlStats {
				sqlStats = false
			} else {
				sqlStats = true
			}

			fmt.Printf("Option [%s] set to [%t]\n", trimmed, sqlStats)
			return
		}

		if trimmed == "!live-stream" {
			if sqlLiveStream {
				sqlLiveStream = false
			} else {
				sqlLiveStream = true
			}

			fmt.Printf("Option [%s] set to [%t]\n", trimmed, sqlLiveStream)
			return
		}

		golog.Errorf("Unknown option [%s]", trimmed)
		return
	}

	finalQ := fmt.Sprintf("%s %s", sqlQuery, sql)
	if sql != "" {
		if strings.HasSuffix(finalQ, ";") {
			validation, err := e.client.ValidateSQL(strings.Replace(finalQ, "  ", " ", 0), 0)

			if err != nil {
				golog.Error(err)
				os.Exit(1)
			}

			var lintError bool
			for _, lint := range validation.Lints {
				lintType := strings.ToLower(lint.Type)
				if lintType == "error" || lintType == "warning" {
					lintError = true
					golog.Errorf("Validation error: [%s]", lint.Text)
				}
			}

			if lintError {
				sqlQuery = ""
				LivePrefixState.LivePrefix = "lenses-sql>"
				LivePrefixState.IsEnable = true
				return
			}

			runSQL(e.interactiveCmd, finalQ, sqlMeta, sqlKeys, sqlKeysOnly, sqlLiveStream, sqlStats)

			file, err := os.Create(e.sqlHistoryPath)
			if err != nil {
				golog.Fatalf("Cannot open file [%s]. [%s]", e.sqlHistoryPath, err.Error())
			}
			defer file.Close()

			_, errS := file.WriteString(sql)

			if errS != nil {
				golog.Fatalf("Error writing history to file [%s]. [%s]", e.sqlHistoryPath, err.Error())
			}

			_, errF := file.WriteString(finalQ)

			if errF != nil {
				golog.Fatalf("Error writing history to file [%s]. [%s]", e.sqlHistoryPath, err.Error())
			}

			sqlQuery = ""
			LivePrefixState.LivePrefix = "lenses-sql>"
			LivePrefixState.IsEnable = true
			return
		}

		sqlQuery = finalQ
		LivePrefixState.LivePrefix = "......... >"
		LivePrefixState.IsEnable = true
	}
	return
}
