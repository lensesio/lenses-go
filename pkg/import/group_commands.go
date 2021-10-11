package imports

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewImportGroupsCommand creates `import groups` command
func NewImportGroupsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "groups",
		Short:            "groups",
		Example:          `import groups --dir groups`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.GroupsPath)
			if err := loadGroups(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to load user groups. [%s]", err.Error())
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

func loadGroups(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading user groups from [%s]", loadpath)
	files, err := utils.FindFiles(loadpath)
	if err != nil {
		return err
	}

	currentGroups, err := client.GetGroups()

	if err != nil {
		return err
	}
	for _, file := range files {

		var group api.Group
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &group); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		found := false
		for _, g := range currentGroups {
			if g.Name == group.Name {
				found = true
				payload := &api.Group{
					Name:                       group.Name,
					Description:                group.Description,
					Namespaces:                 group.Namespaces,
					ScopedPermissions:          group.ScopedPermissions,
					AdminPermissions:           group.AdminPermissions,
					ConnectClustersPermissions: group.ConnectClustersPermissions,
				}

				if err := config.Client.UpdateGroup(payload); err != nil {
					golog.Errorf("Error updating user group [%s]. [%s]", group.Name, err.Error())
					return err
				}
				golog.Infof("Updated group [%s]", group.Name)
			}
		}

		if found {
			continue
		}

		if err := client.CreateGroup(&group); err != nil {
			golog.Errorf("Error creating user group [%s] from [%s] [%s]", group.Name, loadpath, err.Error())
			return err
		}
		golog.Infof("Created user group [%s]", group.Name)

	}

	return nil
}
