package export

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewExportSchemasCmd to export schemas to yaml
func NewExportSchemasCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use: "schemas",
		Long: heredoc.Doc(`
			Export Schemas

			The schemas can be exported to different Kafka Cluster to allow for best GitOps practices.
		`),
		Example: heredoc.Doc(`
			$ lenses-cli export schemas --name="<NAME>"
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			setExecutionMode(client)
			checkFileFlags(cmd)

			err := WriteSchemas(cmd, client, name)
			return errors.WithStack(err)
		},
	}

	cmd.Flags().StringVarP(&landscapeDir, "dir", "D", ".", "Base directory to export to")
	cmd.Flags().StringVarP(&name, "name", "N", "", "Schema Name")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)

	return cmd
}

// WriteSchemas to a file
func WriteSchemas(cmd *cobra.Command, client *api.Client, name string) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	if name != "" {
		return writeSchema(output, client, name)
	}

	subjects, err := client.GetSubjects()
	if err != nil {
		return err
	}
	for _, sub := range subjects {
		if err := writeSchema(output, client, sub.Name); err != nil {
			return err
		}
	}
	return nil
}

func writeSchema(outputFormat string, client *api.Client, name string) error {

	schema, err := client.GetSchema(name)
	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("%s.%s", strings.ToLower(strings.ReplaceAll(schema.Name, " ", "_")), strings.ToLower(outputFormat))
	filePath := filepath.Join(landscapeDir, pkg.SchemasPath, fileName)

	if err := utils.WriteFile(landscapeDir, pkg.SchemasPath, fileName, outputFormat, schema); err != nil {
		return err
	}
	golog.Infof("exported to file '%s'", filePath)

	return nil
}
