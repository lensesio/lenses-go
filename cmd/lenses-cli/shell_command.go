package main

import (
	"fmt"
	"os"
	"bufio"

	"github.com/landoop/lenses-go"
	"github.com/landoop/bite"
	"github.com/spf13/cobra"
	"github.com/c-bata/go-prompt"
	"github.com/kataras/golog"
)

var interactiveCmd *cobra.Command
var interactiveShell bool
var sqlHistoryPath = fmt.Sprintf("%s/history", lenses.DefaultConfigurationHomeDir)

func init() {
	app.AddCommand(newInteractiveCommand())
}

func newInteractiveCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "shell",
		Short:            "shell",
		Example:          `shell`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			interactiveShell = true

			fmt.Printf(`
    __                                 ________    ____
   / /   ___  ____  ________  _____   / ____/ /   /  _/
  / /   / _ \/ __ \/ ___/ _ \/ ___/  / /   / /    / /  
 / /___/  __/ / / (__  )  __(__  )  / /___/ /____/ /   
/_____/\___/_/ /_/____/\___/____/   \____/_____/___/   
Docs at https://docs.lenses.io
Connected to [%s] as [%s], context [%s]
Use "!" to set output options [!keys|!keysOnly|!stats|!meta|!pretty]

`, client.Config.Host, client.User.Name, configManager.config.CurrentContext)

			
			var histories []string

			if _, err := os.Stat(sqlHistoryPath); os.IsExist(err) {
				file, err := os.Open(sqlHistoryPath)
				if err != nil {
					golog.Warnf("Unable to open command history. [%s]", err.Error())
				}
				defer file.Close()

				scanner := bufio.NewScanner(file)
				for scanner.Scan() {
					histories = append(histories, scanner.Text())
				}

				if err := scanner.Err(); err != nil {
					golog.Fatal(err)
				}
			}

			p := prompt.New(
				sqlExecutor,
				sqlCompleter,
				prompt.OptionTitle(fmt.Sprintf("lenses: connected to [%s] ", client.Config.Host)),
				prompt.OptionPrefix("lenses-sql> "),
				prompt.OptionLivePrefix(changeLivePrefix),
				prompt.OptionInputTextColor(prompt.Turquoise),
				prompt.OptionPrefixTextColor(prompt.White),
				prompt.OptionHistory(histories),
			)

			p.Run()

			return nil
		
		},
	}

	interactiveCmd = cmd
	bite.CanPrintJSON(cmd)

	return cmd
}
