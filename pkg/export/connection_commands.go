package export

import (
	"fmt"
	"strings"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportConnectionsCommand creates `export connections`
func NewExportConnectionsCommand() *cobra.Command {
	var connectionName string
	cmd := &cobra.Command{
		Use:   "connections",
		Short: "export connections",
		Example: `export connections
export connections --name connection-name`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			if err := writeConnections(cmd, connectionName); err != nil {
				return fmt.Errorf("error while exporting connections. [%s]", err.Error())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().StringVar(&connectionName, "name", "", "The name of the connection to extract")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

// writeConnections retrieves and writes one or all connections to a file
func writeConnections(cmd *cobra.Command, connectionName string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "writing connections to base directory [%s]\n", landscapeDir)

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	if connectionName != "" {
		connection, err := config.Client.GetConnection(connectionName)
		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("connection-%s-%s.%s", strings.ToLower(strings.ReplaceAll(connection.Name, " ", "_")), connection.Name, strings.ToLower(output))
		return utils.WriteFile(landscapeDir, pkg.ConnectionsFilePath, fileName, output, connection)
	}

	connections, err := config.Client.GetConnections()
	if err != nil {
		return err
	}

	for _, connection := range connections {
		connectionComplete, err := config.Client.GetConnection(connection.Name)
		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("connection-%s-%s.%s", strings.ToLower(strings.ReplaceAll(connection.Name, " ", "_")), connection.Name, strings.ToLower(output))
		err = utils.WriteFile(landscapeDir, pkg.ConnectionsFilePath, fileName, output, connectionComplete)
		if err != nil {
			return fmt.Errorf("could not export connection to file %s", fileName)
		}
	}

	return nil
}
