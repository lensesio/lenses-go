package dataset

import (
	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

const metadataLong = `Description:
  Lenses can store and update the Metadata for a Dataset(Kafka Topic, ES Index). You can 
  use the updateMetadata, to update the Description of a Dataset.

  Be aware, that you need the "UpdateMetadata" permission to execute the command
`

// NewDatasetGroupCmd Group Cmd
func NewDatasetGroupCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "dataset",
		Short: "Use the Dataset Cmd, to execute action for Datasets(Kafka Topics, ES Indices)",
		SilenceErrors:    true,
		TraverseChildren: true,
		Args:  cobra.NoArgs,
	}

	cmd.AddCommand(NewDatasetUpdateMetadataCmd())
	return cmd
}

// NewDatasetUpdateMetadataCmd updates the Dataset Metadata
func NewDatasetUpdateMetadataCmd() *cobra.Command {
	var connection, name, description string

	cmd := &cobra.Command{
		Use: "updateMetadata [CONNECTION] [NAME] [DESCRIPTION]",
		Short: "Manage your metadata for Datasets(i.e: Description)",
		Long: metadataLong,
		SilenceErrors: true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
				if err := config.Client.UpdateMetadata(connection, name, description); err != nil {
				golog.Errorf("Failed to update Lenses Description. [%s]", err.Error())
				return err
			}
			return bite.PrintInfo(cmd, "Lenses Metadata have been updated successfully")
		},
	}

	cmd.Flags().StringVar(&connection, "connection", "", "Name of the connection")
	cmd.Flags().StringVar(&name, "name", "", "Name of the dataset")
	cmd.Flags().StringVar(&description, "description", "", "Description of the dataset")
	cmd.MarkFlagRequired("connection")
	cmd.MarkFlagRequired("name")

	_ = bite.CanBeSilent(cmd)

	return cmd
}