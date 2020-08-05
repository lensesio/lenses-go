package export

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewExportSchemasCommand creates `export schemas` command
func NewExportSchemasCommand() *cobra.Command {
	var name, version string

	cmd := &cobra.Command{
		Use:              "schemas",
		Short:            "export schemas",
		Example:          `export schemas --resource-name my-schema-value --version 1. If no name is supplied the latest versions of all schemas are exported`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			versionInt, err := strconv.Atoi(version)
			if err != nil {
				golog.Errorf("Version [%s] is not at integer", version)
				return err
			}

			if name != "" {
				if err := writeSchema(cmd, config.Client, name, versionInt); err != nil {
					golog.Errorf("Error writing schema. [%s]", err.Error())
					return err
				}
				return nil
			}

			if err := writeSchemas(cmd, config.Client); err != nil {
				golog.Errorf("Error writing schemas. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract dependencies, topics, acls, quotas, alerts")
	cmd.Flags().StringVar(&name, "resource-name", "", "The schema to export. Both the key schema and value schema are exported")
	cmd.Flags().StringVar(&version, "version", "0", "The schema version to export.")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Schemas with the prefix only")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeSchemas(cmd *cobra.Command, client *api.Client) error {

	subjects, err := client.GetSubjects()

	if err != nil {
		return err
	}

	for _, subject := range subjects {
		if prefix != "" && !strings.HasPrefix(subject, prefix) {
			continue
		}

		// don't export control topics
		excluded := false
		for _, exclude := range systemTopicExclusions {
			if strings.HasPrefix(subject, exclude) ||
				strings.Contains(subject, "KSTREAM-") ||
				strings.Contains(subject, "_agg_") ||
				strings.Contains(subject, "_sql_store_") {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		if err := writeSchema(cmd, client, subject, 0); err != nil {
			golog.Error(fmt.Sprintf("Error while exporting schema [%s]", subject))
			return err
		}
	}

	return nil
}

func writeSchema(cmd *cobra.Command, client *api.Client, name string, version int) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	var schema api.Schema
	var err error

	if version != 0 {
		schema, err = client.GetSchemaAtVersion(name, version)
	} else {
		schema, err = client.GetLatestSchema(name)
	}

	pretty, _ := utils.PrettyPrint([]byte(schema.AvroSchema))

	schema.AvroSchema = string(pretty)
	schema.AvroSchema = strings.TrimSpace(schema.AvroSchema)
	schema.AvroSchema = strings.Replace(schema.AvroSchema, "\t", "  ", -1)
	schema.AvroSchema = strings.Replace(schema.AvroSchema, " \n", "\n", -1)

	if err != nil {
		return err
	}

	request := client.GetSchemaAsRequest(schema)
	fileName := fmt.Sprintf("schema-%s.%s", strings.ToLower(name), strings.ToLower(output))
	return utils.WriteFile(landscapeDir, pkg.SchemasPath, fileName, output, request)
}
