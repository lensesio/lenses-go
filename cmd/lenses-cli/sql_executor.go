package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
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

var SqlQuery string

func changeLivePrefix() (string, bool) {
	return LivePrefixState.LivePrefix, LivePrefixState.IsEnable
}

func sqlExecutor(sql string) {
	if strings.HasPrefix(sql, "!") {
		trimmed := strings.Trim(sql, " ")

		if trimmed == "!options" {
			fmt.Printf("Options: keys=%t, keysOnly=%t, meta=%t, stats=%t, live-stream=%t\n", sqlKeys, sqlKeysOnly, sqlMeta, sqlStats, sqlLiveStream)
			return
		}

		if trimmed == "!pretty" {

			if bite.GetJSONPrettyFlag(interactiveCmd) {
				interactiveCmd.Flags().Set("pretty", "false")
			} else {
				interactiveCmd.Flags().Set("pretty", "true")
			}

			fmt.Printf("Option [%s] set to [%t]\n", trimmed, bite.GetJSONPrettyFlag(interactiveCmd))
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

	finalQ := fmt.Sprintf("%s %s", SqlQuery, sql)
	if sql != "" {
		if strings.HasSuffix(finalQ, ";") {
			validation, err := client.ValidateSQL(strings.Replace(finalQ, "  ", " ", 0), 0)

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
				SqlQuery = ""
				LivePrefixState.LivePrefix = "lenses-sql>"
				LivePrefixState.IsEnable = true
				return
			}

			runSQL(interactiveCmd, finalQ, sqlMeta, sqlKeys, sqlKeysOnly, sqlLiveStream, sqlStats)

			file, err := os.Create(sqlHistoryPath)
			if err != nil {
				golog.Fatalf("Cannot open file [%s]. [%s]", sqlHistoryPath, err.Error())
			}
			defer file.Close()

			_, errS := file.WriteString(sql)

			if errS != nil {
				golog.Fatalf("Error writing history to file [%s]. [%s]", sqlHistoryPath, err.Error())
			}

			_, errF := file.WriteString(finalQ)

			if errF != nil {
				golog.Fatalf("Error writing history to file [%s]. [%s]", sqlHistoryPath, err.Error())
			}

			SqlQuery = ""
			LivePrefixState.LivePrefix = "lenses-sql>"
			LivePrefixState.IsEnable = true
			return
		}

		SqlQuery = finalQ
		LivePrefixState.LivePrefix = "......... >"
		LivePrefixState.IsEnable = true
	}
	return
}
