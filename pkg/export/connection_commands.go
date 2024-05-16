package export

import (
	"fmt"
	"hash/fnv"
	"io/fs"
	"path/filepath"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportConnectionsCommand creates `export connections`
func NewExportConnectionsCommand(client apiClient, writer fileWriter) *cobra.Command {
	var connectionName string
	var provisioningVersion int32
	const provisioningFileName = "provisioning.yaml"

	cmd := &cobra.Command{
		Use:              "connections",
		Short:            "export connections",
		Example:          `export connections --name connection-name`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			if provisioningVersion == 1 {
				golog.Warnf("Exporting connectors in the deprecated format. Please add --version 2 for the new format")
				if err := writeConnections(cmd, connectionName, client); err != nil {
					return fmt.Errorf("connections export: %w", err)
				}
				return nil
			} else {
				if connectionName != "" {
					golog.Warn("Exporting a specific connection is not supported in v2. Connetions are exported all at once.")
				}
				if err := writeConnectionsV2(client, provisioningFileName, writer); err != nil {
					return fmt.Errorf("connections export: %w", err)
				}
				return nil
			}
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().StringVar(&connectionName, "name", "", "The name of the connection to extract")
	cmd.Flags().Int32VarP(&provisioningVersion, "version", "v", 1, "version of the yaml output format")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

// writeConnections retrieves and writes one or all connections to a file
func writeConnections(cmd *cobra.Command, connectionName string, client apiClient) error {
	fmt.Fprintf(cmd.OutOrStdout(), "writing connections to base directory [%s]\n", landscapeDir)

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	if connectionName != "" {
		connection, err := client.GetConnection(connectionName)
		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("connection-%s-%s.%s", strings.ToLower(strings.ReplaceAll(connection.Name, " ", "_")), connection.Name, strings.ToLower(output))
		return utils.WriteFile(landscapeDir, pkg.ConnectionsFilePath, fileName, output, connection)
	}

	connections, err := client.GetConnections()
	if err != nil {
		return err
	}

	for _, connection := range connections {
		connectionComplete, err := client.GetConnection(connection.Name)
		if err != nil {
			return err
		}

		// Since connections can differ in case, we use a hash to ensure uniqueness
		// and avoid writing one file over another with the same name
		h := fnv.New64a()
		h.Write([]byte(connection.Name))

		fileName := strings.ToLower(fmt.Sprintf("connection-%s-%08x.%s", connection.Name, uint32(h.Sum64()), output))

		if err = utils.WriteFile(landscapeDir, pkg.ConnectionsFilePath, fileName, output, connectionComplete); err != nil {
			return fmt.Errorf("could not export connection to file %s: %w", fileName, err)
		}
	}

	return nil
}

func writeConnectionsV2(client apiClient, provisioningFileName string, fileW fileWriter) error {
	provisionedConnections, err := client.GetConnectionsState()
	if err != nil {
		return fmt.Errorf("fetch provisioning connections: %w", err)
	}

	//create landscapeDir if it does not exist
	if err := fileW.MkdirAll(landscapeDir, 0o755); err != nil {
		return err
	}

	// merge landscapeDir and filePath
	filePath := filepath.Join(landscapeDir, provisioningFileName)
	golog.Infof("Exporting provisioned connections to %s", filePath)
	err = fileW.WriteFile(filePath, provisionedConnections, 0644)
	if err != nil {
		return err
	}

	return nil
}

type apiClient interface {
	GetConnectionsState() (string, error)
	GetConnection(string) (api.Connection, error)
	GetConnections() ([]api.ConnectionList, error)
}

type fileWriter interface {
	MkdirAll(string, fs.FileMode) error
	WriteFile(string, string, fs.FileMode) error
}
