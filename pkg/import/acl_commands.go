package imports

import (
	"fmt"
	"reflect"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

// NewImportAclsCommand creates `import acls` command
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
	files, err := utils.FindFiles(loadpath)
	if err != nil {
		return err
	}

	knownACLs, err := client.GetACLs()

	if err != nil {
		return err
	}

	for _, file := range files {
		var candidateACLs []api.ACL
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &candidateACLs); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		var imported bool
		// Import only new ACLs
	ImportACLs:
		for _, candidateACL := range candidateACLs {
			for _, knownACL := range knownACLs {
				if reflect.DeepEqual(knownACL, candidateACL) {
					continue ImportACLs
				}
			}

			if err := client.CreateOrUpdateACL(candidateACL); err != nil {
				return fmt.Errorf("error creating/updating acl from [%s] [%s]", loadpath, err.Error())
			}
			fmt.Fprintf(cmd.OutOrStdout(), "imported ACL [%s] successfully\n", candidateACL)

			imported = true
		}

		importFilePath := fmt.Sprintf("%s/%s", loadpath, file.Name())
		if !imported {
			fmt.Fprintf(cmd.OutOrStdout(), "no new ACLs have been found for import from %s\n", importFilePath)
		}
	}
	return nil
}
