package dataset

import (
	"errors"
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

const metadataLong = `Description:
  Lenses can store a user-defined description for a Dataset (i.e. Kafka topics, ES indices).
  Be aware, that you need the "UpdateMetadata" permission to execute the command
`

// NewDatasetGroupCmd Group Cmd
func NewDatasetGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "dataset",
		Short:            "Use the dataset command to set user-defined metadata on Kafka topics and ES indices",
		SilenceErrors:    true,
		TraverseChildren: true,
		Args:             cobra.NoArgs,
	}

	cmd.AddCommand(ListDatasetsCmd())
	cmd.AddCommand(UpdateDatasetDescriptionCmd())
	cmd.AddCommand(UpdateDatasetTagsCmd())
	cmd.AddCommand(RemoveDatasetDescriptionCmd())
	cmd.AddCommand(RemoveDatasetTagsCmd())
	return cmd
}

// listDatasetsOutput is a common denominator of the various dataset objects
// sent to the output, modelled after the UI.
type listDatasetsOutput struct {
	Name       string      `header:"name"`
	Size       interface{} `header:"size"`
	Records    interface{} `header:"records"`
	DataSource string      `header:"data source"`
}

// ListDatasetsCmd defines the cobra command to list datasets.
func ListDatasetsCmd() *cobra.Command {
	var max int
	var query string
	records := newEnumFlag(api.RecordCountAll, api.RecordCountEmpty, api.RecordCountNonEmpty)
	var connections []string

	cmd := &cobra.Command{
		Use:              "list",
		Short:            "Lists the datasets",
		Example:          `dataset list --connections kafka --records empty`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			params := api.ListDatasetsParameters{
				RecordCount: records.optPtr(),
				Connections: connections,
			}
			if query != "" {
				params.Query = &query
			}
			res, err := config.Client.ListDatasetsPg(params, max)
			if err != nil {
				return err
			}
			var os []listDatasetsOutput
			for _, ds := range res {
				var o listDatasetsOutput
				switch v := ds.(type) {
				case api.Elastic:
					o = listDatasetsOutput{
						Name:       v.Name,
						Size:       derefOrNA(v.SizeBytes),
						Records:    derefOrNA(v.Records),
						DataSource: string(api.SourceTypeElastic),
					}
				case api.Kafka:
					o = listDatasetsOutput{
						Name:       v.Name,
						Size:       derefOrNA(v.SizeBytes),
						Records:    derefOrNA(v.Records),
						DataSource: string(api.SourceTypeKafka),
					}
					if v.IsCompacted { // Hide total number of records if compacted; the UI does this as well.
						o.Records = derefOrNA(nil)
					}
				case api.Postgres:
					o = listDatasetsOutput{
						Name:       v.Name,
						Size:       derefOrNA(v.SizeBytes),
						Records:    derefOrNA(v.Records),
						DataSource: string(api.SourceTypePostgres),
					}
				case api.SchemaRegistrySubject:
					o = listDatasetsOutput{
						Name:       v.Name,
						Size:       derefOrNA(v.SizeBytes),
						Records:    derefOrNA(v.Records),
						DataSource: string(api.SourceTypeSchemaRegistrySubject),
					}
				default:
					return fmt.Errorf("unknown type: %T", ds)
				}
				os = append(os, o)
			}
			return bite.PrintObject(cmd, os)
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "A search keyword to match dataset, fields and description against.")
	cmd.Flags().IntVar(&max, "max", 0, "Maximum number of results to return.")
	cmd.Flags().Var(&records, "records", "Filter the amount of records. Allowed values: "+strings.Join(records.allowedValues(), ", ")+".")
	cmd.Flags().StringSliceVar(&connections, "connections", nil, "Connection names to filter by. All connections will be included when no value is supplied.")

	return cmd
}

// UpdateDatasetDescriptionCmd updates the Dataset Metadata
func UpdateDatasetDescriptionCmd() *cobra.Command {
	var connection, name, description string

	cmd := &cobra.Command{
		Use:              "update-description [CONNECTION] [NAME]",
		Short:            "Set a dataset description",
		Long:             metadataLong,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(strings.TrimSpace(description)) == 0 {
				err := errors.New("--description value cannot be blank")
				golog.Errorf("Failed to update dataset description. [%s]", err.Error())
				return err
			}

			if err := config.Client.UpdateDatasetDescription(connection, name, description); err != nil {
				golog.Errorf("Failed to update dataset description. [%s]", err.Error())
				return err
			}
			return bite.PrintInfo(cmd, "Dataset description has been updated successfully")
		},
	}

	cmd.Flags().StringVar(&connection, "connection", "", "Name of the connection")
	cmd.Flags().StringVar(&name, "name", "", "Name of the dataset")
	cmd.Flags().StringVar(&description, "description", "", "Description of the dataset")
	cmd.MarkFlagRequired("description")
	cmd.MarkFlagRequired("connection")
	cmd.MarkFlagRequired("name")

	_ = bite.CanBeSilent(cmd)

	return cmd
}

// UpdateDatasetTagsCmd updates the Dataset Metadata
func UpdateDatasetTagsCmd() *cobra.Command {
	var connection, name string
	var tags []string

	cmd := &cobra.Command{
		Use:   "update-tags [CONNECTION] [NAME]",
		Short: "Set a dataset tags",
		Example: `
		dataset update-tags --connection kafka \
		           --name mytopic \
				   --tag t1 \
				   --tag t2
		`,
		Long:             metadataLong,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(tags) == 0 {
				return errors.New("Tags cannot be empty")
			}

			if err := config.Client.UpdateDatasetTags(connection, name, tags); err != nil {
				return err
			}
			return bite.PrintInfo(cmd, "Dataset tags have been updated successfully")
		},
	}

	cmd.Flags().StringVar(&connection, "connection", "", "Name of the connection")
	cmd.Flags().StringVar(&name, "name", "", "Name of the dataset")
	cmd.Flags().StringArrayVar(&tags, "tag", []string{}, "tag assigned to the connection, can be defined multiple times")
	cmd.MarkFlagRequired("connection")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("tag")

	_ = bite.CanBeSilent(cmd)

	return cmd
}

// RemoveDatasetDescriptionCmd unsets a dataset description
func RemoveDatasetDescriptionCmd() *cobra.Command {
	var connection, name string

	cmd := &cobra.Command{
		Use:              "remove-description [CONNECTION] [NAME] [DESCRIPTION]",
		Short:            "Unsets a dataset description",
		Long:             metadataLong,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			//Setting the description to empty string will result in the field being omitted from the submitted JSON
			//which the backend will handle by unsetting the description record (see `UpdateDatasetDescription`'s
			//`omitempty` annotation)
			if err := config.Client.UpdateDatasetDescription(connection, name, ""); err != nil {
				return err
			}
			return bite.PrintInfo(cmd, "Dataset description has been removed")
		},
	}

	cmd.Flags().StringVar(&connection, "connection", "", "Name of the connection")
	cmd.Flags().StringVar(&name, "name", "", "Name of the dataset")
	cmd.MarkFlagRequired("connection")
	cmd.MarkFlagRequired("name")

	_ = bite.CanBeSilent(cmd)
	return cmd
}

// RemoveDatasetTagsCmd unsets a dataset description
func RemoveDatasetTagsCmd() *cobra.Command {
	var connection, name string

	cmd := &cobra.Command{
		Use:              "remove-tags [CONNECTION] [NAME]",
		Short:            "Remove all tags associated to a dataset",
		Long:             metadataLong,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Client.UpdateDatasetTags(connection, name, []string{}); err != nil {
				return err
			}
			return bite.PrintInfo(cmd, "Dataset tags have been removed")
		},
	}

	cmd.Flags().StringVar(&connection, "connection", "", "Name of the connection")
	cmd.Flags().StringVar(&name, "name", "", "Name of the dataset")
	cmd.MarkFlagRequired("connection")
	cmd.MarkFlagRequired("name")

	_ = bite.CanBeSilent(cmd)
	return cmd
}

func derefOrNA(i *int) interface{} {
	if i == nil {
		return "N/A"
	}
	return *i
}

// enumFlag is a flag that can only get assigned values that are in the set of
// allowed values. It implements pflag.Value.
type enumFlag[T ~string] struct {
	allowed []T // The set of possible values the flag can assume.
	value   T   // The value assigned to it after Set()ting it.
}

func newEnumFlag[T ~string](vs ...T) enumFlag[T] {
	return enumFlag[T]{allowed: vs}
}

func (e *enumFlag[T]) String() string { return string(e.value) }
func (e *enumFlag[T]) Set(v string) error {
	for _, a := range e.allowed {
		if a == T(v) {
			e.value = T(v)
			return nil
		}
	}
	return fmt.Errorf("allowed are %s; not: %q", strings.Join(e.allowedValues(), ", "), v)
}
func (e *enumFlag[T]) Type() string { return "string" }
func (e *enumFlag[T]) optPtr() *T {
	if e.value == "" {
		return nil
	}
	return &e.value
}
func (e *enumFlag[T]) allowedValues() (ss []string) {
	for _, v := range e.allowed {
		ss = append(ss, string(v))
	}
	return
}
