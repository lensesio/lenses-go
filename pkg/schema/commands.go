package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

type schemaView struct {
	ID            int             `json:"id" header:"ID,text"`
	Name          string          `json:"name" header:"Name"`
	LatestVersion int             `json:"latest_version" header:"Latest /"`
	Versions      []int           `json:"versions" header:"All Versions"`
	Avro          json.RawMessage `json:"schema"` // only for json output.
}

//NewSchemasGroupCommand creates `schemas` command
func NewSchemasGroupCommand() *cobra.Command {
	var unwrap bool
	root := &cobra.Command{
		Use:           "schemas",
		Short:         "List all available schemas",
		Example:       "schemas",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

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

				schema, err := newSchemaView(client, sc, !tableMode)
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
					// remove the last \r from the below fmt.Printf.
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

			// Author's note for future contributors, do not confuse about this style:
			// this will block until we finish with the schemas goroutine -> see close(errors) above.
			for err := range errors {
				errOL++
				errBody.WriteString(fmt.Sprintf("[%s%d]. [%v]\n", strings.Repeat(" ", 2), errOL, err))
			}

			if errBody.Len() > 0 {
				if tableMode {
					return fmt.Errorf("\nErrors raised during the operation:\n[%s]", errBody.String())
				}
				return fmt.Errorf(errBody.String())
			}

			return nil
		},
	}

	root.Flags().BoolVar(&unwrap, "unwrap", false, "prints only the names as a list of strings separated by line endings")
	bite.CanPrintJSON(root)
	root.AddCommand(NewGlobalCompatibilityLevelGroupCommand())

	return root
}

//NewGlobalCompatibilityLevelGroupCommand creates `schemas compatibility` command
func NewGlobalCompatibilityLevelGroupCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "compatibility [?set [compatibility]]",
		Short:            "Get the global compatibility level",
		Example:          `compatibility`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			lv, err := config.Client.GetGlobalCompatibilityLevel()
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), string(lv))
			return nil
		},
	}

	rootSub.AddCommand(NewUpdateGlobalCompatibilityLevelCommand())
	return rootSub

}

//NewUpdateGlobalCompatibilityLevelCommand creates `schemas set` command
func NewUpdateGlobalCompatibilityLevelCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "set",
		Short:         "Change the global compatibility level",
		Example:       `compatibility set FULL`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) != 1 {
				return fmt.Errorf("Compatibility value is required")
			}
			lv := args[0]
			if !api.IsValidCompatibilityLevel(lv) {
				return fmt.Errorf("Compatibility value is not valid, use one of those: [%s]", joinValidCompatibilityLevels(", "))
			}

			if err := config.Client.UpdateGlobalCompatibilityLevel(api.CompatibilityLevel(lv)); err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Global compatibility level updated")
		},
	}

	bite.CanBeSilent(cmd)

	return cmd
}

//NewSchemaGroupCommand creates `schema` command
func NewSchemaGroupCommand() *cobra.Command {
	var (
		name               string
		versionStringOrInt string
		id                 int
	)

	root := &cobra.Command{
		Use:              "schema",
		Short:            "Manage particular schema based on its name, get a schema based on the ID or register a new one",
		Example:          `schema --id=1 or schema --name="name" [flags] or schema register --name="name" --avro="..."`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			if id > 0 {
				if err := getSchemaByID(cmd, id, client); err != nil {
					golog.Errorf("Failed to retrieve schema for id [%d]. [%s]", id, err.Error())
					return err
				}
			}

			// from below and after, the name flag is required.
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}

			// it's not empty, always, so it's called latest.
			if versionStringOrInt != "" {
				// golog.Errorf("Failed to retrieve schema [%s], version [%s]", name, versionStringOrInt)
				return getSchemaByVersion(cmd, client, name, versionStringOrInt)
			}

			return nil
		},
	}

	// get the schema based on its name.
	root.Flags().StringVar(&name, "name", "", `Lookup by schema name`)
	// get the schema based on its id.
	root.Flags().IntVar(&id, "id", 0, "Lookup by schema id")
	// it's not required, the default is "latest", get a schema based on a specific version.
	root.Flags().StringVar(&versionStringOrInt, "version", api.SchemaLatestVersion, "Latest or numeric value lookup schema based on a specific  version")
	// if true then the schema will be NOT printed with indent.
	bite.CanPrintJSON(root)

	// subcommands.
	root.AddCommand(NewRegisterSchemaCommand())
	root.AddCommand(NewGetSchemaVersionsCommand())
	root.AddCommand(NewDeleteSchemaCommand())
	root.AddCommand(NewDeleteSchemaVersionCommand())
	root.AddCommand(NewSchemaCompatibilityLevelGroupCommand()) // includes subcommands.

	return root
}

//NewRegisterSchemaCommand creates `schema register` command
func NewRegisterSchemaCommand() *cobra.Command {
	var schema api.Schema

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

			id, err := config.Client.RegisterSchema(schema.Name, schema.AvroSchema)
			if err != nil {
				golog.Errorf("Failed to register schema [%s]. [%s]", schema.Name, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Registered schema [%s] with id [%d]", schema.Name, id)
		},
	}

	cmd.Flags().StringVar(&schema.Name, "name", "", `Schema name`)
	cmd.Flags().StringVar(&schema.AvroSchema, "avro", schema.AvroSchema, "Avro schema")

	bite.Prepend(cmd, bite.FileBind(&schema))
	bite.CanBeSilent(cmd)

	return cmd
}

//NewGetSchemaVersionsCommand creates `schema versions` command
func NewGetSchemaVersionsCommand() *cobra.Command {
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

			versions, err := config.Client.GetSubjectVersions(name)
			if err != nil {
				golog.Errorf("Failed to retrieve schema [%s]. [%s]", name, err.Error())
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

//NewDeleteSchemaCommand creates `schema delete` command
func NewDeleteSchemaCommand() *cobra.Command {
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

			deletedVersions, err := config.Client.DeleteSubject(name)
			if err != nil {
				golog.Errorf("Failed to delete schema [%s]. [%s]", name, err.Error())
				return err
			}

			if bite.ExpectsFeedback(cmd) {
				// return printJSON(cmd, outlineIntResults("version", deletedVersions))
				return bite.PrintObject(cmd, bite.OutlineIntResults(cmd, "version", deletedVersions))
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `Schema name to delete`)
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)

	return cmd
}

//NewDeleteSchemaVersionCommand creates `schema delete-version` command
func NewDeleteSchemaVersionCommand() *cobra.Command {
	var name, versionStringOrInt string

	cmd := &cobra.Command{
		Use:           "delete-version",
		Short:         "Delete a specific version of the schema registered under this name. This command only deletes the version and the schema id remains intact making it still possible to decode data using the schema id. Returns the version of the deleted schema",
		Example:       `schema delete-version --name="name" --version="latest or numeric"`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

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
				bite.FriendlyError(cmd, pkg.ErrResourceNotFoundMessage, "Unable to delete schema with version [%s]. Either schema [%s] or version [%s] does not exist.", versionStringOrInt, name, versionStringOrInt)
				return err
			}

			return bite.PrintInfo(cmd, "[%d]", deletedVersion)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `Schema name`)
	cmd.Flags().StringVar(&versionStringOrInt, "version", api.SchemaLatestVersion, "Latest or numeric value schema version")
	bite.CanBeSilent(cmd)

	return cmd
}

//NewSchemaCompatibilityLevelGroupCommand creates `schema compatibility` command
func NewSchemaCompatibilityLevelGroupCommand() *cobra.Command {
	var name string
	rootSub := &cobra.Command{
		Use:              "compatibility [?set [compatibility]]",
		Short:            "Print or change the compatibility level of a schema",
		Example:          `schema compatibility --name="name" or schema compatibility set FULL --name="name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}

			lv, err := config.Client.GetSubjectCompatibilityLevel(name)
			if err != nil {
				bite.FriendlyError(cmd, pkg.ErrResourceNotFoundMessage, "Unable to retrieve the compatibility level, subject [%s] either does not exist or no compatibility has been set", name)
				return err
			}

			return bite.PrintInfo(cmd, string(lv))
		},
	}

	rootSub.Flags().StringVar(&name, "name", "", `Schema name`)
	bite.CanBeSilent(rootSub)

	rootSub.AddCommand(NewUpdateSchemaCompatibilityLevelCommand())

	return rootSub
}

//NewUpdateSchemaCompatibilityLevelCommand creates `schema compatibility set` command
func NewUpdateSchemaCompatibilityLevelCommand() *cobra.Command {
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
			if !api.IsValidCompatibilityLevel(lv) {
				return fmt.Errorf("compatibility value is not valid, use one of those [%s]", joinValidCompatibilityLevels(", "))
			}

			if err := config.Client.UpdateSubjectCompatibilityLevel(name, api.CompatibilityLevel(lv)); err != nil {
				bite.FriendlyError(cmd, pkg.ErrResourceNotFoundMessage, "unable to change the compatibility level of the schema, schema [%s] does not exist", name)
				return err
			}

			return bite.PrintInfo(cmd, "Compatibility level for [%s] updated", name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `Schema name`)
	bite.CanBeSilent(cmd)

	return cmd
}

func newSchemaView(client *api.Client, sc api.Schema, withAvro bool) (schemaView, error) {
	versions, err := client.GetSubjectVersions(sc.Name)
	if err != nil {
		return schemaView{}, err
	}

	schema := schemaView{ID: sc.ID, Name: sc.Name, LatestVersion: sc.Version, Versions: versions}

	if withAvro {
		schema.Avro, err = api.JSONAvroSchema(sc.AvroSchema)
		if err != nil {
			return schema, err
		}
	}

	return schema, nil
}

func getSchemaByID(cmd *cobra.Command, id int, client *api.Client) error {
	result, err := client.GetSchema(id)
	if err != nil {
		return err
	}

	schemaRawJSON, err := api.JSONAvroSchema(result)
	if err != nil {
		return err
	}

	// return printJSON(cmd, schemaRawJSON)
	return bite.PrintJSON(cmd, schemaRawJSON)
}

// the only valid version string is the "latest"
// so try to take the number and manage SchemaAtVersion otherwise LatestSchema.
func latestOrInt(name, versionStringOrInt string, str func(versionString string) error, num func(version int) error) error {
	// the only valid version string is the "latest"
	// so try to take the number and manage SchemaAtVersion otherwise LatestSchema.
	if versionStringOrInt != api.SchemaLatestVersion {
		version, err := strconv.Atoi(versionStringOrInt)
		if err != nil {
			return err
		}
		return num(version)
	}

	return str(api.SchemaLatestVersion)
}

func getSchemaByVersion(cmd *cobra.Command, client *api.Client, name, versionStringOrInt string) error {

	readSchema := func(versionStringOrInt string) (schema api.Schema, err error) {
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

	schema, err := newSchemaView(client, sc, !bite.ExpectsFeedback(cmd))
	if err != nil {
		return err
	}

	return bite.PrintObject(cmd, schema)
}

func joinValidCompatibilityLevels(sep string) string {
	var b strings.Builder
	end := len(api.ValidCompatibilityLevels) - 1
	for i, lv := range api.ValidCompatibilityLevels {
		if i == end {
			b.WriteString(string(lv))
			break
		}

		b.WriteString(string(lv))
		b.WriteString(sep)
	}

	return b.String()
}
