package shell

import (
	"bufio"
	"fmt"
	"os"

	"github.com/c-bata/go-prompt"
	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/sql"
	"github.com/spf13/cobra"
)

var sqlHistoryPath = fmt.Sprintf("%s/history", api.DefaultConfigurationHomeDir)

//NewInteractiveCommand creates `shell` command
func NewInteractiveCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "shell",
		Short:            "shell",
		Example:          `shell`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			sql.InteractiveShell = true

			fmt.Printf(`
    __                                 ________    ____
   / /   ___  ____  ________  _____   / ____/ /   /  _/
  / /   / _ \/ __ \/ ___/ _ \/ ___/  / /   / /    / /  
 / /___/  __/ / / (__  )  __(__  )  / /___/ /____/ /   
/_____/\___/_/ /_/____/\___/____/   \____/_____/___/   
Docs at https://docs.lenses.io
Connected to [%s] as [%s], context [%s]
Use "!" to set output options [!keys|!keysOnly|!stats|!meta|!pretty]
Crtl+D to exit

`, client.Config.Host, client.User.Name, config.Manager.Config.CurrentContext)

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
			executor := sql.NewExecutor(cmd, client, sqlHistoryPath)

			p := prompt.New(
				executor.Execute,
				sql.Completer,
				prompt.OptionTitle(fmt.Sprintf("lenses: connected to [%s] ", client.Config.Host)),
				prompt.OptionPrefix("lenses-sql> "),
				prompt.OptionLivePrefix(executor.ChangeLivePrefix),
				prompt.OptionInputTextColor(prompt.Turquoise),
				prompt.OptionPrefixTextColor(prompt.White),
				prompt.OptionHistory(histories),
			)

			p.Run()

			return nil

		},
	}
	bite.CanPrintJSON(cmd)

	return cmd
}
