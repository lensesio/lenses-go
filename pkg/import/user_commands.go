package imports

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewImportUsersCommand creates `import users` command
func NewImportUsersCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "users",
		Short:            "users",
		Example:          `import users --dir users`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.UsersPath)
			if err := loadUsers(config.Client, cmd, path); err != nil {
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

func loadUsers(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading users from [%s]", loadpath)
	files := utils.FindFiles(loadpath)

	currentUsers, err := client.GetUsers()

	if err != nil {
		return err
	}
	for _, file := range files {

		var users []api.UserMember
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &users); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}
		for _, u := range users {
			found := false
			for _, cu := range currentUsers {
				if cu.Username == u.Username {
					found = true
				}
			}
			if found {
				continue
			}
			if err := client.CreateUser(&u); err != nil {
				golog.Errorf("Error creating user [%s] from [%s] [%s]", u.Username, loadpath, err.Error())
				return err
			}
		}
	}
	golog.Infof("Created/updated users from [%s]", loadpath)

	return nil
}
