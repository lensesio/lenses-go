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

var acl api.ACL

//NewImportAclsCommand creates `import acls` command
func NewImportAclsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "acls",
		Example:          `import acls --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.AclsPath)
			if err := loadAcls(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to load acls. [%s]", err.Error())
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

func loadAcls(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading acls from [%s]", loadpath)
	files := utils.FindFiles(loadpath)

	lacls, err := client.GetACLs()

	if err != nil {
		return err
	}

	for _, file := range files {
		var acls []api.ACL
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &acls); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		found := true
		for _, l := range lacls {
			if acl.Host == l.Host &&
				acl.Operation == l.Operation &&
				acl.PermissionType == l.PermissionType &&
				acl.Principal == l.Principal &&
				acl.ResourceName == l.ResourceName &&
				acl.ResourceType == l.ResourceType {
				found = true
			}
		}

		if found {
			continue
		}

		for _, acl := range acls {
			if err := client.CreateOrUpdateACL(acl); err != nil {
				golog.Errorf("Error creating/updating acl from [%s] [%s]", loadpath, err.Error())
				return err
			}
		}

		golog.Infof("Created/updated ACLs from [%s]", loadpath)
	}
	return nil
}
