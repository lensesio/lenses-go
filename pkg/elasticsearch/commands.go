package elasticsearch

import (
	"github.com/landoop/bite"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

// IndexesCommand displays available indexes
func IndexesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "elasticsearch-indexes",
		Short:         "List all available elasticsearch indexes",
		Example:       "elasticsearch-indexes",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			indexes, err := client.GetIndexes()

			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, indexes)
		},
	}

	return cmd
}

// IndexCommand displays index data
func IndexCommand() *cobra.Command {
	var connectionName string
	var indexName string

	cmd := &cobra.Command{
		Use:           "elasticsearch-index",
		Short:         "List all available elasticsearch indexes",
		Example:       `elasticsearch-index --connection="connection" --name="index"`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			index, err := client.GetIndex(connectionName, indexName)

			if err != nil {
				return err
			}

			indexview := MakeIndexView(index)

			return bite.PrintObject(cmd, indexview)
		},
	}

	cmd.Flags().StringVar(&connectionName, "connection", "", "Connection to use")
	cmd.Flags().StringVar(&indexName, "name", "", "Index to look for")

	return cmd
}
