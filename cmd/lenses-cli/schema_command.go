package main

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newSchemasGroupCommand())
	rootCmd.AddCommand(newSchemaGroupCommand())
}

func newSchemasGroupCommand() *cobra.Command {
	rootCmd := cobra.Command{
		Use:           "schemas",
		Short:         "List all available schemas",
		Example:       exampleString("schemas"),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			subjects, err := client.GetSubjects()
			if err != nil {
				return err
			}

			return printJSON(cmd.OutOrStdout(), outlineStringResults("name", subjects))
		},
	}

	rootCmd.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	rootCmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	rootCmd.AddCommand(newGlobalCompatibilityLevelGroupCommand())

	return &rootCmd
}

func newGlobalCompatibilityLevelGroupCommand() *cobra.Command {
	rootSub := cobra.Command{
		Use:              "compatibility [?set [compatibility]]",
		Short:            "Get the global compatibility level",
		Example:          exampleString(`compatibility`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lv, err := client.GetGlobalCompatibilityLevel()
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(lv))
			return nil
		},
	}

	rootSub.AddCommand(newUpdateGlobalCompatibilityLevelCommand())
	return &rootSub

}

func newUpdateGlobalCompatibilityLevelCommand() *cobra.Command {

	cmd := cobra.Command{
		Use:           "set",
		Short:         "Change the global compatibility level",
		Example:       exampleString(`compatibility set FULL`),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("compatibility value is required")
			}
			lv := args[0]
			if !lenses.IsValidCompatibilityLevel(lv) {
				return fmt.Errorf("compatibility value is not valid, use one of those: %s", joinValidCompatibilityLevels(", "))
			}

			if err := client.UpdateGlobalCompatibilityLevel(lenses.CompatibilityLevel(lv)); err != nil {
				return err
			}

			return echo(cmd, "Global compatibility level updated")
		},
	}

	return &cmd
}

// --id=
// --name="..." --version=1
// --name="..." == --name="..." --version="latest"
// register --name="..." --avro="..."
func newSchemaGroupCommand() *cobra.Command {
	var (
		name               string
		versionStringOrInt string
		id                 int
	)

	root := cobra.Command{
		Use:              "schema",
		Short:            "Work with a particular schema based on its name, get a schema based on the ID or register a new one",
		Example:          exampleString(`schema --id=1 or schema --name="name" [flags] or schema register --name="name" --avro="..."`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id > 0 {
				return getSchemaByID(cmd, id)
			}

			// from below and after, the name flag is required.
			if err := checkRequiredFlags(cmd, flags{"name": name}); err != nil {
				return err
			}

			// it's not empty, always, so it's called latest.
			if versionStringOrInt != "" {
				return getSchemaByVersion(cmd, name, versionStringOrInt, !noPretty)
			}

			return nil
		},
	}

	// get the schema based on its name.
	root.Flags().StringVar(&name, "name", "", `--name="name" lookup by schema name`)
	// get the schema based on its id.
	root.Flags().IntVar(&id, "id", 0, "--id=42 lookup by schema id")
	// it's not required, the default is "latest", get a schema based on a specific version.
	root.Flags().StringVar(&versionStringOrInt, "version", lenses.SchemaLatestVersion, "--version=latest or numeric value lookup schema based on a specific  version")
	// if true then the schema will be NOT printed with indent.
	root.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	root.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	// subcommands.
	root.AddCommand(newRegisterSchemaCommand())
	root.AddCommand(newGetSchemaVersionsCommand())
	root.AddCommand(newDeleteSchemaCommand())
	root.AddCommand(newDeleteSchemaVersionCommand())
	root.AddCommand(newSchemaCompatibilityLevelGroupCommand()) // includes subcommands.

	return &root
}

func getSchemaByID(cmd *cobra.Command, id int) error {
	result, err := client.GetSchema(id)
	if err != nil {
		return err
	}

	schemaRawJSON, err := lenses.JSONAvroSchema(result)
	if err != nil {
		return err
	}
	return printJSON(cmd.OutOrStderr(), schemaRawJSON)
}

// the only valid version string is the "latest"
// so try to take the number and work with SchemaAtVersion otherwise LatestSchema.
func latestOrInt(name, versionStringOrInt string, str func(versionString string) error, num func(version int) error) error {
	// the only valid version string is the "latest"
	// so try to take the number and work with SchemaAtVersion otherwise LatestSchema.
	if versionStringOrInt != lenses.SchemaLatestVersion {
		version, err := strconv.Atoi(versionStringOrInt)
		if err != nil {
			return err
		}
		return num(version)
	}

	return str(lenses.SchemaLatestVersion)
}

func getSchemaByVersion(cmd *cobra.Command, name, versionStringOrInt string, pretty bool) error {

	readSchema := func(versionStringOrInt string) (schema lenses.Schema, err error) {
		err = latestOrInt(name, versionStringOrInt, func(_ string) error {
			schema, err = client.GetLatestSchema(name)
			return err
		}, func(version int) error {
			schema, err = client.GetSchemaAtVersion(name, version)
			return err
		})

		return
	}

	schema, err := readSchema(versionStringOrInt)
	if err != nil {
		return err
	}

	rawJSONSchema, err := lenses.JSONAvroSchema(schema.AvroSchema)
	if err != nil {
		return err
	}

	return printJSON(cmd.OutOrStdout(), struct {
		lenses.Schema
		JSONSchema json.RawMessage `json:"schema"`
	}{schema, rawJSONSchema})
}

func newRegisterSchemaCommand() *cobra.Command {
	var schema lenses.Schema

	cmd := cobra.Command{
		Use:              "register",
		Short:            "Register a new schema under a particular name and print the new schema identifier",
		Example:          exampleString(`schema register --name="name" --avro="..."`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// load from file.
				if err := loadFile(cmd, args[0], &schema); err != nil {
					return err
				}
			}

			if err := checkRequiredFlags(cmd, flags{"name": schema.Name, "avro": schema.AvroSchema}); err != nil {
				return err
			}

			id, err := client.RegisterSchema(schema.Name, schema.AvroSchema)
			if err != nil {
				return err
			}

			return echo(cmd, "Registered schema %s with id %d", schema.Name, id)
		},
	}

	cmd.Flags().StringVar(&schema.Name, "name", "", `--name="name"`)
	cmd.Flags().StringVar(&schema.AvroSchema, "avro", schema.AvroSchema, "--avro=")

	return &cmd
}

func newGetSchemaVersionsCommand() *cobra.Command {
	var name string

	cmd := cobra.Command{
		Use:           "versions",
		Short:         "List all versions of a particular schema",
		Example:       exampleString(`schema --name="name" versions`),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": name}); err != nil {
				return err
			}

			versions, err := client.GetSubjectVersions(name)
			if err != nil {
				return err
			}

			return printJSON(cmd.OutOrStdout(), outlineIntResults("version", versions))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `--name="name"`)
	cmd.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	cmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	return &cmd
}

func newDeleteSchemaCommand() *cobra.Command {
	var name string

	cmd := cobra.Command{
		Use:           "delete",
		Short:         "Delete a schema",
		Example:       exampleString(`schema delete --name="name"`),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": name}); err != nil {
				return err
			}

			deletedVersions, err := client.DeleteSubject(name)
			if err != nil {
				return err
			}

			if !silent {
				return printJSON(cmd.OutOrStdout(), outlineIntResults("version", deletedVersions))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `--name="name"`)
	cmd.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	cmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	return &cmd
}

func newDeleteSchemaVersionCommand() *cobra.Command {
	var name, versionStringOrInt string

	cmd := cobra.Command{
		Use:           "delete-version",
		Short:         "Delete a specific version of the schema registered under this name. This command only deletes the version and the schema id remains intact making it still possible to decode data using the schema id. Returns the version of the deleted schema",
		Example:       exampleString(`schema delete-version --name="name" --version="latest or numeric"`),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": name}); err != nil {
				return err
			}

			if len(args) < 1 {
				return fmt.Errorf("version command is missing")
			}

			var (
				err            error
				deletedVersion int
			)

			err = latestOrInt(name, versionStringOrInt, func(_ string) error {
				deletedVersion, err = client.DeleteLatestSubjectVersion(name)
				return err
			}, func(version int) error {
				deletedVersion, err = client.DeleteSubjectVersion(name, version)
				return err
			})

			if err != nil {
				return err
			}

			return echo(cmd, "%d", deletedVersion)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `--name="name"`)
	cmd.Flags().StringVar(&versionStringOrInt, "version", lenses.SchemaLatestVersion, "--version=latest or numeric value")

	return &cmd
}

func joinValidCompatibilityLevels(sep string) string {
	var b strings.Builder
	end := len(lenses.ValidCompatibilityLevels) - 1
	for i, lv := range lenses.ValidCompatibilityLevels {
		if i == end {
			b.WriteString(string(lv))
			break
		}

		b.WriteString(string(lv))
		b.WriteString(sep)
	}

	return b.String()
}

func newSchemaCompatibilityLevelGroupCommand() *cobra.Command {
	var name string
	rootSub := cobra.Command{
		Use:              "compatibility [?set [compatibility]]",
		Short:            "Print or change the compatibility level of a schema",
		Example:          exampleString(`schema --name="name" compatibility or compatibility set FULL`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": name}); err != nil {
				return err
			}

			lv, err := client.GetSubjectCompatibilityLevel(name)
			if err != nil {
				return err
			}

			return echo(cmd, string(lv))
		},
	}

	rootSub.Flags().StringVar(&name, "name", "", `--name="name"`)

	rootSub.AddCommand(newUpdateSchemaCompatibilityLevelCommand())
	return &rootSub
}

func newUpdateSchemaCompatibilityLevelCommand() *cobra.Command {
	var name string

	cmd := cobra.Command{
		Use:           "set",
		Short:         "Change compatibility level of a schema",
		Example:       exampleString(`schema --name="name" compatibility set FULL`),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": name}); err != nil {
				return err
			}

			if len(args) != 1 {
				return fmt.Errorf("compatibility value is required")
			}

			lv := args[0]
			if !lenses.IsValidCompatibilityLevel(lv) {
				return fmt.Errorf("compatibility value is not valid, use one of those %s", joinValidCompatibilityLevels(", "))
			}

			if err := client.UpdateSubjectCompatibilityLevel(name, lenses.CompatibilityLevel(lv)); err != nil {
				return err
			}

			return echo(cmd, "Compatibility level for %s updated", name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `--name="name"`)
	return &cmd
}
