package imports

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewImportSchemasCommand creates `import schemas` command
func NewImportSchemasCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "schemas",
		Short:            "schemas",
		Example:          `import schemas --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.SchemasPath)
			if err := loadSchemas(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to load schemas. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func loadSchemas(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading schemas from [%s]", loadpath)
	files := utils.FindFiles(loadpath)

	for _, file := range files {
		var schema api.SchemaAsRequest
		if err := load(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &schema); err != nil {
			return err
		}

		_, err := client.RegisterSchema(schema.Name, schema.AvroSchema)

		if err != nil {
			golog.Errorf("Error creating schema from file [%s]. [%s]", loadpath, err.Error())
			return err
		}

		golog.Infof("Created schema from [%s]", loadpath)
	}

	return nil
}
