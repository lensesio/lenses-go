package export

import (
	"errors"
	"fmt"
	"hash/fnv"
	"os"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportConnectorsCommand creates `export connectors` command
func NewExportConnectorsCommand() *cobra.Command {
	var name, cluster string
	var yamlVersion int32

	cmd := &cobra.Command{
		Use:              "connectors",
		Short:            "export connectors",
		Example:          `export connectors --version 2 --resource-name my-connector --cluster-name cluster1`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client
			setExecutionMode(client)
			checkFileFlags(cmd)
			if yamlVersion != 2 {
				golog.Warnf("Exporting connectors in the deprecated format. Please add --version 2 for the new format")
			}
			if err := writeConnectors(cmd, client, cluster, name, yamlVersion); err != nil {
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
	cmd.Flags().Int32Var(&yamlVersion, "version", 1, "Which export version to use, default is 1(deprecated), 2 for the new format")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

// writeConnectors writes the connectors to files as yaml
// If a clusterName is provided the connectors are filtered by clusterName
// If a name is provided the connectors are filtered by connector name
// When version is 2, the connectors are written in the new format geared towards the gitops roadmap
func writeConnectors(cmd *cobra.Command, client *api.Client, clusterName string, name string, yamlVersion int32) error {
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

			// Since connectors can differ in case, we use a hash to ensure uniqueness
			// and avoid writing one file over another with the same name
			h := fnv.New64a()
			h.Write([]byte(connectorName))

			output := strings.ToUpper(bite.GetOutPutFlag(cmd))
			fileName := strings.ToLower(fmt.Sprintf("connector-%s-%s-%08x.%s", cluster, connectorName, uint32(h.Sum64()), output))

			if yamlVersion == 2 {
				err = writeVersion2(client, connectorName, cluster, fileName)
				if err != nil {
					return err
				}
			} else {
				err = writeDeprecatedFormat(cmd, client, connectorName, cluster, fileName)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func writeVersion2(client *api.Client, connector string, cluster string, file string) error {
	connectorAsCode, err := client.GetConnectorAsCode(cluster, connector)
	if err != nil {
		return errors.New("Failed to get connector: " + connector + " in the connect-cluster:" + cluster)
	}

	//create landscapeDir if it does not exist
	if _, err := os.Stat(landscapeDir); os.IsNotExist(err) {
		err = os.Mkdir(landscapeDir, 0755)
		if err != nil {
			return errors.New("Failed to create directory: " + landscapeDir)
		}
	}
	// merge landscapeDir and filePath
	filePath := fmt.Sprintf("%s/%s", landscapeDir, file)
	f, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	golog.Infof("Exporting connector [%s.%s] to [%s]", cluster, connector, filePath)
	_, err = f.WriteString(connectorAsCode)
	if err != nil {
		return errors.New("Failed to write connector: " + connector + " in the connect-cluster:" + cluster + " to the file: " + file)
	}

	return nil
}

func writeDeprecatedFormat(cmd *cobra.Command, client *api.Client, connectorName string, cluster string, fileName string) error {
	connector, err := client.GetConnector(cluster, connectorName)
	if err != nil {
		return err
	}

	if connector.Config[connectorClassKey] == sqlConnectorClass {
		return nil
	}
	request := connector.ConnectorAsRequest()

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	if output == "TABLE" {
		output = "YAML"
	}

	golog.Debugf("Exporting connector [%s.%s] to [%s%s]", cluster, connectorName, landscapeDir, fileName)
	if err := utils.WriteFile(landscapeDir, pkg.ConnectorsPath, fileName, output, request); err != nil {
		return err
	}

	if dependents {
		return handleDependents(cmd, client, fmt.Sprintf("%s:%s", connector.ClusterName, connector.Name))
	}
	return nil
}
