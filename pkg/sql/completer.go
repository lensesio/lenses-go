package sql

import (
	"fmt"
	"os"
	"strings"

	"github.com/c-bata/go-prompt"
	"github.com/kataras/golog"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
)

func checkValidation(validation api.SQLValidationResponse) bool {
	for _, lint := range validation.Lints {
		var val = strings.ToLower(lint.Type)
		if val == "error" || val == "warning" {
			for _, innerLint := range validation.Lints {
				golog.Errorf("sError: text [%s]", innerLint.Text)
			}
			return false
		}
	}

	return true
}

//Completer sql completer
func Completer(d prompt.Document) []prompt.Suggest {

	if strings.HasPrefix(d.GetWordBeforeCursor(), "!") {
		return prompt.FilterHasPrefix(optionSuggestions(), d.GetWordBeforeCursor(), true)
	}

	sql := fmt.Sprintf("%s%s", sqlQuery, d.CurrentLine())
	caret := d.CursorPositionCol() + len(sqlQuery)

	keywords, err := config.Client.ValidateSQL(strings.Replace(sql, "  ", " ", 0), caret)
	if err != nil {
		golog.Error(err)
		os.Exit(1)
	}

	if d.TextBeforeCursor() == "" {
		return []prompt.Suggest{}
	}

	var suggestions []prompt.Suggest

	for _, s := range keywords.Suggestions {
		suggestions = append(suggestions, prompt.Suggest{Text: s.Display, Description: s.Text})
	}

	return prompt.FilterHasPrefix(suggestions, d.GetWordBeforeCursor(), true)
}

func optionSuggestions() []prompt.Suggest {
	return []prompt.Suggest{
		{Text: "!keys", Description: "Toggle printing message keys"},
		{Text: "!keys-only", Description: "Toggle printing keys only from message, no value"},
		{Text: "!live-stream", Description: "Toggle continuous query mode"},
		{Text: "!meta", Description: "Toggle printing message metadata"},
		{Text: "!stats", Description: "Toggle printing query stats"},
		{Text: "!options", Description: "Print current options"},
		{Text: "!pretty", Description: "Toggle pretty printing query output"},
	}
}
