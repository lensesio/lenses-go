package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/landoop/lenses-go"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newSchemasGroupCommand())
	app.AddCommand(newSchemaGroupCommand())
}

type schemaView struct {
	ID            int             `json:"id" header:"ID,text"`
	Name          string          `json:"name" header:"Name"`
	LatestVersion int             `json:"latest_version" header:"Latest /"`
	Versions      []int           `json:"versions" header:"All Versions"`
	Avro          json.RawMessage `json:"schema"` // only for json output.
}

func newSchemaView(sc lenses.Schema, withAvro bool) (schemaView, error) {
	versions, err := client.GetSubjectVersions(sc.Name)
	if err != nil {
		return schemaView{}, err
	}

	schema := schemaView{ID: sc.ID, Name: sc.Name, LatestVersion: sc.Version, Versions: versions}

	if withAvro {
		schema.Avro, err = lenses.JSONAvroSchema(sc.AvroSchema)
		if err != nil {
			return schema, err
		}
	}

	return schema, nil
}

func newSchemasGroupCommand() *cobra.Command {
	var unwrap bool

	root := &cobra.Command{
		Use:           "schemas",
		Short:         "List all available schemas",
		Example:       "schemas",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			subjects, err := client.GetSubjects()
			if err != nil {
				return err
			}

			sort.Strings(subjects)

			if unwrap {
				for _, name := range subjects {
					fmt.Fprintln(cmd.OutOrStdout(), name)
				}
				return nil
			}

			// Author's note: if you ever change this code please re-run it with go build -race && ./lenses-cli schemas.
			getSchemaDetails := func(subject string, tableMode bool, schemas chan<- schemaView, errors chan<- error, wg *sync.WaitGroup) {
				defer wg.Done()

				sc, err := client.GetLatestSchema(subject)
				if err != nil {
					errors <- err
					return
				}

				schema, err := newSchemaView(sc, !tableMode)
				if err != nil {
					errors <- err
					return
				}

				schemas <- schema
			}

			var (
				schemas = make(chan schemaView)
				errors  = make(chan error, len(subjects))
			)

			wg := new(sync.WaitGroup)
			wg.Add(len(subjects))

			tableMode := bite.ExpectsFeedback(cmd)

			// collect schemas in their own context.
			go func() {
				var (
					totalSchemas []schemaView
					proceeds     uint64
					total        = len(subjects)
				)

				for sch := range schemas {
					// note that we need the completed list of schemas
					// in order to have all table features for all rows that may be useful for end-users,
					// so we can't just print the incoming, we must wait to finish all fetch operations.
					totalSchemas = append(totalSchemas, sch)

					proceeds++
					// move two columns forward,
					// try to avoid first-pos blinking,
					// blinking only when the number changes, the text shows 1 col after the bar,
					// and when finish
					// reposition of the cursor to the beginning and clean the view, so table can be rendered
					// without any join headers.
					//
					// How to debug the order of proceeds:
					// comment the line after wg.Wait(): fmt.Fprintf(os.Stdout, "\n\033[1A\033[K")
					// remove the last \r from the below fmt.Pritnf.
					fmt.Fprintf(os.Stdout, "\033[2C%d/%d\r", proceeds, total)
				}

				// remove the prev line(the processing current/total line) so we can show a clean table or errors.
				fmt.Fprintf(os.Stdout, "\n\033[1A\033[K")

				if err := bite.PrintObject(cmd, totalSchemas); err != nil {
					errors <- err
				}

				close(errors)
			}()

			for _, subject := range subjects {
				go getSchemaDetails(subject, tableMode, schemas, errors, wg)
			}

			wg.Wait()

			// close the schemas but not the errors yet, after all schemas we must make sure that we can still retrieve errors
			// from the bite.PrintObject, although is extremely rare to error there.
			close(schemas)

			// collect any errors.
			var (
				errBody = new(bytes.Buffer)
				errOL   int
			)

			for err := range errors {
				errOL++
				errBody.WriteString(fmt.Sprintf("%s%d. %v\n", strings.Repeat(" ", 2), errOL, err))
			}

			if errBody.Len() > 0 {
				if tableMode {
					return fmt.Errorf("\nErrors raised during the operation:\n%s", errBody.String())
				}
				return fmt.Errorf(errBody.String())
			}

			return nil
		},
	}

	root.Flags().BoolVar(&unwrap, "unwrap", false, "prints only the names as a list of strings separated by line endings")
	bite.CanPrintJSON(root)
	root.AddCommand(newGlobalCompatibilityLevelGroupCommand())

	return root
}

func newGlobalCompatibilityLevelGroupCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "compatibility [?set [compatibility]]",
		Short:            "Get the global compatibility level",
		Example:          `compatibility`,
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
	return rootSub

}

func newUpdateGlobalCompatibilityLevelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "set",
		Short:         "Change the global compatibility level",
		Example:       `compatibility set FULL`,
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

			return bite.PrintInfo(cmd, "Global compatibility level updated")
		},
	}

	bite.CanBeSilent(cmd)

	return cmd
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

	root := &cobra.Command{
		Use:              "schema",
		Short:            "Work with a particular schema based on its name, get a schema based on the ID or register a new one",
		Example:          `schema --id=1 or schema --name="name" [flags] or schema register --name="name" --avro="..."`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id > 0 {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "schema with id: %d does not exist", id)
				return getSchemaByID(cmd, id)
			}

			// from below and after, the name flag is required.
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}

			// it's not empty, always, so it's called latest.
			if versionStringOrInt != "" {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "schema with name: '%s' and version: '%s' does not exist", name, versionStringOrInt)
				return getSchemaByVersion(cmd, name, versionStringOrInt)
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
	bite.CanPrintJSON(root)

	// subcommands.
	root.AddCommand(newRegisterSchemaCommand())
	root.AddCommand(newGetSchemaVersionsCommand())
	root.AddCommand(newDeleteSchemaCommand())
	root.AddCommand(newDeleteSchemaVersionCommand())
	root.AddCommand(newSchemaCompatibilityLevelGroupCommand()) // includes subcommands.

	return root
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

	// return printJSON(cmd, schemaRawJSON)
	return bite.PrintJSON(cmd, schemaRawJSON)
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

func getSchemaByVersion(cmd *cobra.Command, name, versionStringOrInt string) error {

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

	sc, err := readSchema(versionStringOrInt)
	if err != nil {
		return err
	}

	schema, err := newSchemaView(sc, !bite.ExpectsFeedback(cmd))
	if err != nil {
		return err
	}

	return bite.PrintObject(cmd, schema)
}

func newRegisterSchemaCommand() *cobra.Command {
	var schema lenses.Schema

	cmd := &cobra.Command{
		Use:              "register",
		Short:            "Register a new schema under a particular name and print the new schema identifier",
		Example:          `schema register --name="name" --avro="..."`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": schema.Name, "avro": schema.AvroSchema}); err != nil {
				return err
			}

			id, err := client.RegisterSchema(schema.Name, schema.AvroSchema)
			if err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Registered schema %s with id %d", schema.Name, id)
		},
	}

	cmd.Flags().StringVar(&schema.Name, "name", "", `--name="name"`)
	cmd.Flags().StringVar(&schema.AvroSchema, "avro", schema.AvroSchema, "--avro=")

	bite.Prepend(cmd, bite.FileBind(&schema))
	bite.CanBeSilent(cmd)

	return cmd
}

func newGetSchemaVersionsCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:           "versions",
		Short:         "List all versions of a particular schema",
		Example:       `schema versions --name="name"`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}

			versions, err := client.GetSubjectVersions(name)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "schema with name: '%s` does not exist", name)
				return err
			}

			// return printJSON(cmd, outlineIntResults("version", versions))
			return bite.PrintObject(cmd, bite.OutlineIntResults(cmd, "version", versions))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `--name="name"`)

	bite.CanPrintJSON(cmd)

	return cmd
}

func newDeleteSchemaCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:           "delete",
		Short:         "Delete a schema",
		Example:       `schema delete --name="name"`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}

			deletedVersions, err := client.DeleteSubject(name)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "schema with name: '%s` does not exist", name)
				return err
			}

			if bite.ExpectsFeedback(cmd) {
				// return printJSON(cmd, outlineIntResults("version", deletedVersions))
				return bite.PrintObject(cmd, bite.OutlineIntResults(cmd, "version", deletedVersions))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `--name="name"`)
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)

	return cmd
}

func newDeleteSchemaVersionCommand() *cobra.Command {
	var name, versionStringOrInt string

	cmd := &cobra.Command{
		Use:           "delete-version",
		Short:         "Delete a specific version of the schema registered under this name. This command only deletes the version and the schema id remains intact making it still possible to decode data using the schema id. Returns the version of the deleted schema",
		Example:       `schema delete-version --name="name" --version="latest or numeric"`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
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
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to delete the schema with version '%s', schema %s does not exist", versionStringOrInt, name)
				return err
			}

			return bite.PrintInfo(cmd, "%d", deletedVersion)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `--name="name"`)
	cmd.Flags().StringVar(&versionStringOrInt, "version", lenses.SchemaLatestVersion, "--version=latest or numeric value")
	bite.CanBeSilent(cmd)

	return cmd
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
	rootSub := &cobra.Command{
		Use:              "compatibility [?set [compatibility]]",
		Short:            "Print or change the compatibility level of a schema",
		Example:          `schema --name="name" compatibility or compatibility set FULL`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}

			lv, err := client.GetSubjectCompatibilityLevel(name)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to retrieve the compatibility level, subject '%s' does not exist", name)
				return err
			}

			return bite.PrintInfo(cmd, string(lv))
		},
	}

	rootSub.Flags().StringVar(&name, "name", "", `--name="name"`)
	bite.CanBeSilent(rootSub)

	rootSub.AddCommand(newUpdateSchemaCompatibilityLevelCommand())

	return rootSub
}

func newUpdateSchemaCompatibilityLevelCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:           "set",
		Short:         "Change compatibility level of a schema",
		Example:       `schema --name="name" compatibility set FULL`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
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
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to change the compatibility level of the schema, schema '%s' does not exist", name)
				return err
			}

			return bite.PrintInfo(cmd, "Compatibility level for %s updated", name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `--name="name"`)
	bite.CanBeSilent(cmd)

	return cmd
}
