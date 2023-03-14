package imports

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"

	"github.com/pkg/errors"

	"github.com/MakeNowJust/heredoc"
	"github.com/lensesio/bite"
	"github.com/spf13/cobra"
)

// NewImportSchemasCmd to read schemas from files
func NewImportSchemasCmd() *cobra.Command {
	var path string
	var name string

	cmd := &cobra.Command{
		Use: "schemas",
		Long: heredoc.Doc(`
		Import Schemas

		The settings can be import from a different Kafka Cluster to allow for best GitOps practices.
		`),
		Example: heredoc.Doc(`
		$ lenses-cli import schemas --dir /directory
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path = fmt.Sprintf("%s/%s/", path, pkg.SchemasPath)
			err := ReadSchemas(config.Client, cmd, path)
			return errors.Wrapf(err, "Failed to read schemas")
		},
	}

	cmd.Flags().StringVarP(&path, "dir", "D", ".", "Base directory to import")
	cmd.Flags().StringVarP(&name, "name", "N", "", "Imported Schema Name")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")

	return cmd
}

// ReadSchemas to read the files and import one by one
func ReadSchemas(client *api.Client, cmd *cobra.Command, filePath string) error {
	files, err := utils.FindFiles(filePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		var schema api.WriteSchemaReq
		var fileName = file.Name()

		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", filePath, file.Name()), &schema); err != nil {
			return errors.Wrapf(err, "Could not load file [%s]", fileName)
		}

		schemaName := strings.TrimSuffix(fileName, filepath.Ext(fileName))

		if err := client.WriteSchema(schemaName, schema); err != nil {
			return errors.Wrapf(err, "Could not import Schemas [%s]", fileName)
		}
		golog.Infof("imported schema from file '%s'", filePath)
	}

	return nil
}
