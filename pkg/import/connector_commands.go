package imports

import (
	"fmt"
	"reflect"
	"strconv"
	"time"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/matryer/try"
	"github.com/spf13/cobra"
)

//NewImportConnectorsCommand create `import connectors`
func NewImportConnectorsCommand() *cobra.Command {
	var path string
	var interval string
	var retries int

	cmd := &cobra.Command{
		Use:              "connectors",
		Short:            "connectors",
		Example:          `import connectors --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.ConnectorsPath)
			if err := loadConnectors(config.Client, cmd, path, interval, retries); err != nil {
				return fmt.Errorf("failed to load connectors. [%s]", err.Error())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")
	cmd.Flags().StringVar(&interval, "interval", "0s", "Time between importing two connectors")
	cmd.Flags().IntVar(&retries, "retries", 5, "Number of HTTP retries before exiting")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func loadConnectors(client *api.Client, cmd *cobra.Command, loadpath, interval string, retries int) error {
	intervalDuration, err := time.ParseDuration(interval)
	if err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "loading connectors from [%s]\n", loadpath)
	files, err := utils.FindFiles(loadpath)
	if err != nil {
		return err
	}

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
				var c api.Connector
				err := try.Do(func(attempt int) (bool, error) {
					var err error
					c, err = client.GetConnector(connector.ClusterName, connector.Name)
					if err != nil {
						time.Sleep(intervalDuration)
					}
					return attempt < retries, err
				})
				if err != nil {
					return err
				}

				if !reflect.DeepEqual(c.Config, connector.Config) {
					err := try.Do(func(attempt int) (bool, error) {
						var err error
						_, err = client.UpdateConnector(connector.ClusterName, connector.Name, connector.Config)
						if err != nil {
							fmt.Fprintf(cmd.OutOrStdout(), "failed to update connector '%s' [attempt num. %s]\n", connector.Name, strconv.Itoa(attempt))
							time.Sleep(intervalDuration)
						}
						return attempt < retries, err
					})
					if err != nil {
						return err
					}

					fmt.Fprintf(cmd.OutOrStdout(), "updated connector config for cluster [%s], connector [%s]\n", connector.ClusterName, connector.Name)

					if err != nil {
						return err
					}
				}

				existsOrUpdated = true
				break
			}
		}

		if existsOrUpdated {
			continue
		}

		err = try.Do(func(attempt int) (bool, error) {
			var err error
			_, err = client.CreateConnector(connector.ClusterName, connector.Name, connector.Config)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "failed to create connector '%s' [attempt num. %s]\n", connector.Name, strconv.Itoa(attempt))
				time.Sleep(intervalDuration)
			}
			return attempt < retries, err
		})
		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "created  connector [%s] successfully!\n", connector.Name)
		time.Sleep(intervalDuration)
	}

	return nil
}
