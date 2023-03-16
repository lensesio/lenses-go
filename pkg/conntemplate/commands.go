package conntemplate

import (
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	cobra "github.com/spf13/cobra"
)

// NewConnectionTemplateGroupCommand creates `connection-templates` command
func NewConnectionTemplateGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "connection-templates",
		Short: `List the connection templates`,
		Example: `
connection-templates
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			connectionTemplates, err := config.Client.GetConnectionTemplates()
			if err != nil {
				golog.Errorf("Failed to retrieve connection templates. [%s]", err.Error())
				return err
			}

			outputFlagValue := strings.ToUpper(bite.GetOutPutFlag(cmd))
			if outputFlagValue != "JSON" && outputFlagValue != "YAML" {
				bite.PrintInfo(cmd, "Info: use JSON or YAML output to get the complete object\n\n")
			}

			return bite.PrintObject(cmd, connectionTemplates)
		},
	}

	bite.CanPrintJSON(cmd)

	return cmd
}
