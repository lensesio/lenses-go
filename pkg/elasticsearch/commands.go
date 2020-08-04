package elasticsearch

import (
	"fmt"

	"github.com/landoop/bite"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

// IndexesCommand displays available indexes
func IndexesCommand() *cobra.Command {
	var connectionName string
	var includeSystemIndexes bool
	cmd := &cobra.Command{
		Use:           "elasticsearch-indexes",
		Short:         "List all available elasticsearch indexes",
		Example:       `elasticsearch-indexes --connection="es-default" --include-system-indexes`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			indexes, err := client.GetIndexes(connectionName, includeSystemIndexes)

			if err != nil {
				return fmt.Errorf("Failed to retrieve indexes. Error: [%s]", err.Error())
			}

			return bite.PrintObject(cmd, indexes)
		},
	}

	cmd.Flags().StringVar(&connectionName, "connection", "", "Connection to use")
	cmd.Flags().BoolVar(&includeSystemIndexes, "include-system-indexes", false, "Show system indexes")

	bite.CanPrintJSON(cmd)
	return cmd
}

// IndexCommand displays index data
func IndexCommand() *cobra.Command {
	var connectionName string
	var indexName string

	cmd := &cobra.Command{
		Use:           "elasticsearch-index",
		Short:         "Fetch an elasticsearch index",
		Example:       `elasticsearch-index --connection="es-default" --name="index"`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			index, err := client.GetIndex(connectionName, indexName)

			if err != nil {
				return fmt.Errorf("Failed to retrieve index. Error: [%s]", err.Error())
			}

			indexview := MakeIndexView(index)

			return bite.PrintObject(cmd, indexview)
		},
	}

	cmd.Flags().StringVar(&connectionName, "connection", "", "Connection to use")
	cmd.Flags().StringVar(&indexName, "name", "", "Index to look for")

	bite.CanPrintJSON(cmd)

	return cmd
}
