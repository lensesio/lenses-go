package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportConnectorsCommand creates `export connectors` command
func NewExportConnectorsCommand() *cobra.Command {
	var name, cluster string

	cmd := &cobra.Command{
		Use:              "connectors",
		Short:            "export connectors",
		Example:          `export connectors --resource-name my-connector --cluster-name cluster1`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client
			setExecutionMode(client)
			checkFileFlags(cmd)
			if err := writeConnectors(cmd, client, cluster, name); err != nil {
				golog.Errorf("Error writing connectors. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract dependencies, topics, acls, quotas, alerts")
	cmd.Flags().StringVar(&name, "resource-name", "", "The resource name to export")
	cmd.Flags().StringVar(&cluster, "cluster-name", "", "Select by cluster name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Connector with the prefix in the name only")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

// writeConnectors writes the connectors to files as yaml
// If a clusterName is provided the connectors are filtered by clusterName
// If a name is provided the connectors are filtered by connector name
func writeConnectors(cmd *cobra.Command, client *api.Client, clusterName string, name string) error {
	clusters, err := client.GetConnectClusters()

	if err != nil {
		return err
	}

	for _, cluster := range clusters {

		connectorNames, err := client.GetConnectors(cluster)
		if err != nil {
			golog.Error(err)
			return err
		}

		if clusterName != "" && cluster != clusterName {
			continue
		}

		for _, connectorName := range connectorNames {

			if name != "" && connectorName != name {
				continue
			}

			if prefix != "" && !strings.HasPrefix(connectorName, prefix) {
				continue
			}

			connector, err := client.GetConnector(cluster, connectorName)
			if err != nil {
				return err
			}

			if connector.Config[connectorClassKey] == sqlConnectorClass {
				continue
			}

			request := connector.ConnectorAsRequest()

			output := strings.ToUpper(bite.GetOutPutFlag(cmd))
			fileName := fmt.Sprintf("connector-%s-%s.%s", strings.ToLower(cluster), strings.ToLower(connectorName), strings.ToLower(output))

			if output == "TABLE" {
				output = "YAML"
			}

			golog.Debugf("Exporting connector [%s.%s] to [%s%s]", cluster, connectorName, landscapeDir, fileName)
			if err := utils.WriteFile(landscapeDir, pkg.ConnectorsPath, fileName, output, request); err != nil {
				return err
			}

			if dependents {
				handleDependents(cmd, client, fmt.Sprintf("%s:%s", connector.ClusterName, connector.Name))
			}
		}
	}
	return nil
}
