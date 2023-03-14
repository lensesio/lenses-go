package imports

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewImportConnectionsCommand creates `import connections` command
func NewImportConnectionsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "connections",
		Short:            "Import from a directory named connections",
		Example:          `import connections --dir lenses_export`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.ConnectionsFilePath)
			if err := loadConnections(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to import connections. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import from")

	bite.CanPrintJSON(cmd)
	_ = bite.CanBeSilent(cmd)
	return cmd
}

func loadConnections(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading connections from [%s]", loadpath)

	currentConnections, err := client.GetConnections()
	if err != nil {
		return err
	}

	files, err := utils.FindFiles(loadpath)
	if err != nil {
		return err
	}
	connTemplates, err := config.Client.GetConnectionTemplates()
	if err != nil {
		golog.Errorf("Error getting connection templates [%s]", err.Error())
		return err
	}

	for _, file := range files {
		var connection api.Connection
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &connection); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		found := false
		for _, currentConn := range currentConnections {
			if currentConn.Name == connection.Name {
				found = true
				golog.Infof("Updating connection [%s]", connection.Name)
				if err := config.Client.UpdateConnection(currentConn.Name, connection.Name, "", connection.Configuration, connection.Tags); err != nil {
					golog.Errorf("Error updating connection [%s]. [%s]", connection.Name, err.Error())
					return err
				}
				golog.Infof("Updated connection [%s]", connection.Name)
				continue
			}
		}
		if !found {
			golog.Infof("Creating new connection [%s]", file.Name())
			var connTemplateName string
			for _, connTemplate := range connTemplates {
				if connTemplate.Name == connection.TemplateName {
					connTemplateName = connTemplate.Name
					break
				}
			}
			if connTemplateName == "" {
				golog.Errorf("Connection template %s for connection %s not found [%s]", connection.TemplateName, connection.Name, err.Error())
				return err
			}
			if err := config.Client.CreateConnection(connection.Name, connTemplateName, "", connection.Configuration, connection.Tags); err != nil {
				golog.Errorf("Error creating connection [%s] from [%s] [%s]", connection.Name, loadpath, err.Error())
				return err
			}
			golog.Infof("Created connection [%s]", connection.Name)
		}
	}

	return nil
}
