package schemas

import (
	"fmt"
	"os"

	"github.com/MakeNowJust/heredoc"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

func NewSchemasCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "schema-registry",
		Long: heredoc.Doc(`
			Use this command to operate on various aspects of the
			Schema Registry. You can:

			- View an "AVRO" or "PROTOBUF" Schema.
			- Create or Update a particular Schema.
			- Delete a "Schema" or a "Version".
			- Set the Schema "Compatibility".
			- Set the Default "Compatibility".
		`),
		Example: heredoc.Doc(`
		$ lenses-cli schema-registry
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	rootCmd.AddCommand(NewViewSchemaCmd())
	rootCmd.AddCommand(WriteSchemaCmd())
	rootCmd.AddCommand(SetSchemaCompatibility())
	rootCmd.AddCommand(SetGlobalCompatibility())
	rootCmd.AddCommand(RemoveSchemaVersion())
	rootCmd.AddCommand(RemoveSchema())

	return rootCmd
}

func NewViewSchemaCmd() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use: "get",
		Long: heredoc.Doc(`
			Read the details of a particular Schema. Provide a "name"
			for each Schema.

			Information include "schema", "version", "format". Please wrap
			the name parameter, in quotes to ensure proper encoding support.
		`),
		Example: heredoc.Doc(`
			$ lenses-cli schema-registry get --name="<ΝΑΜΕ>"
		`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client
			schema, err := client.GetV2Schema(name)

			if err != nil {
				return errors.Wrap(err, utils.RED("✘ Error"))
			}
			return bite.PrintJSON(cmd, schema)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, utils.Green("✓ Request succeeded!"))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `Schema Name`)
	cmd.MarkFlagRequired("name")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)

	return cmd
}

func WriteSchemaCmd() *cobra.Command {
	var request api.WriteSchemaReq
	var name string

	cmd := &cobra.Command{
		Use: "write",
		Long: heredoc.Doc(`
		Create a "Schema" if it doesn't exist. Update if it exists.

		Set the "Schema" for either "Avro" or "Protobuf" formats. 
		If no format is provide, Avro will be used as default.

		Note, that proper "Schema" encoding is necessary for both
		"AVRO" or "PROTOBUF"
		`),
		Example: heredoc.Doc(`
		$ lenses-cli schema-regstiry write --name="<NAME>" --format="<FORMAT>" --schema="<SCHEMA>"
		`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client
			err := client.WriteSchema(name, request)
			return errors.Wrap(err, utils.RED("✘ Error"))
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, utils.Green("✓ Request succeeded!"))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Schema Name")
	cmd.Flags().StringVar(&request.Format, "format", "", "Schema Format, either one of 'AVRO', 'PROTOBUF'")
	cmd.Flags().StringVar(&request.Schema, "schema", "", "Schema")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("schema")

	return cmd
}

func SetSchemaCompatibility() *cobra.Command {
	var request api.SetSchemaCompatibilityReq
	var name string

	cmd := &cobra.Command{
		Use: "set",
		Long: heredoc.Doc(`
		Override the Schema Compatibility, with a different one.
		
		Options: "BACKWARDS", "FORWARDS", "NONE", "FULL", "FULL_TRANSITIVE"
		"BACKWARDS_TRANSITIVE", "FORWARDS_TRANSITIVE"
		`),
		Example: heredoc.Doc(`
		$ lenses-cli schema-registry set --name="<NAME>" --compatibility="<COMPATIBILITY>"
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client
			err := client.SetSchemaCompatibility(name, request)

			return errors.Wrap(err, utils.RED("✘ Error"))
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, utils.Green("✓ Request succeeded!"))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Schema Name")
	cmd.Flags().StringVar(&request.Compatibility, "compatibility", "", "Schema Compatibility")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("compatibility")

	return cmd
}

func SetGlobalCompatibility() *cobra.Command {
	var request api.SetGlobalCompatibilityReq

	cmd := &cobra.Command{
		Use: "default",
		Long: heredoc.Doc(`
		Override the Schema Registry Compatibility.
		
		Options: "BACKWARDS", "FORWARDS", "NONE", "FULL", "FULL_TRANSITIVE"
		"BACKWARDS_TRANSITIVE", "FORWARDS_TRANSITIVE"
		`),
		Example: heredoc.Doc(`
		$ lenses-cli schema-registry default --compatibility="<COMPATIBILITY>"
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client
			err := client.SetGlobalCompatibility(request)

			return errors.Wrap(err, utils.RED("✘ Error"))
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, utils.Green("✓ Request succeeded!"))
		},
	}

	cmd.Flags().StringVar(&request.Compatibility, "compatibility", "", "Schema Compatibility")

	cmd.MarkFlagRequired("compatibility")

	return cmd
}

func RemoveSchemaVersion() *cobra.Command {
	var name string
	var version string

	cmd := &cobra.Command{
		Use: "remove-version",
		Long: heredoc.Doc(`
		Remove a specific version of a Schema. You can keep the Schema
		but remove a specific version of it.

		Note, that this will perform a soft removal of the Schema. Not a permanent one.
		`),
		Example: heredoc.Doc(`
		$ lenses-cli schema-registry remove-version --name="<NAME>" --version="<VERSION>"
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client
			err := client.RemoveSchemaVersion(name, version)

			return errors.Wrap(err, utils.RED("✘ Error"))
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, utils.Green("✓ Request succeeded!"))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Schema Name")
	cmd.Flags().StringVar(&version, "version", "", "Schema Version")

	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("version")

	return cmd
}

func RemoveSchema() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use: "remove-schema",
		Long: heredoc.Doc(`
		Remove a Schema and all its versions. Note, that this will perform a soft removal
		of the Schema. Not a permanent one.
		`),
		Example: heredoc.Doc(`
		$ lenses-cli schema-registry remove-schema --name="<NAME>"
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client
			err := client.RemoveSchema(name)

			return errors.Wrap(err, utils.RED("✘ Error"))
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, utils.Green("✓ Request succeeded!"))
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Schema Name")

	cmd.MarkFlagRequired("name")

	return cmd
}
