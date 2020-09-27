package imports

import (
	"fmt"
	"reflect"
	"time"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewImportConnectorsCommand create `import connectors`
func NewImportConnectorsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "connectors",
		Short:            "connectors",
		Example:          `import processors --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.ConnectorsPath)
			if err := loadConnectors(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to load connectors. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func loadConnectors(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading connectors from [%s]", loadpath)
	files := utils.FindFiles(loadpath)

	for _, file := range files {
		var connector api.CreateUpdateConnectorPayload
		if err := load(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &connector); err != nil {
			return err
		}

		connectors, err := client.GetConnectors(connector.ClusterName)

		if err != nil {
			return err
		}

		existsOrUpdated := false
		for _, name := range connectors {
			if name == connector.Name {
				c, err := client.GetConnector(connector.ClusterName, connector.Name)

				if err != nil {
					return err
				}

				if !reflect.DeepEqual(c.Config, connector.Config) {
					_, errU := client.UpdateConnector(connector.ClusterName, connector.Name, connector.Config)
					if errU != nil {
						golog.Errorf("Error updating connector from file [%s]. [%s]", loadpath, errU.Error())
						return errU
					}

					golog.Infof("Updated connector config for cluster [%s], connector [%s]", connector.ClusterName, connector.Name)
				}

				existsOrUpdated = true
				break
			}
		}

		if existsOrUpdated {
			continue
		}
		_, errC := client.CreateConnector(connector.ClusterName, connector.Name, connector.Config)

		if errC != nil {
			golog.Errorf("Error creating connector from file [%s]. [%s]", loadpath, errC.Error())
			return err
		}

		golog.Infof("Created/updated connector from [%s]", loadpath)
		time.Sleep(10 * time.Second)
	}

	return nil
}
