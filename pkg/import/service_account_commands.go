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

//NewImportServiceAccountsCommand creates `import serviceaccounts` command
func NewImportServiceAccountsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "serviceaccounts",
		Short:            "serviceaccounts",
		Example:          `import serviceaccounts --dir users`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.ServiceAccountsPath)
			if err := loadServiceAccounts(config.Client, cmd, path); err != nil {
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

func loadServiceAccounts(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading service accounts from [%s]", loadpath)
	files := utils.FindFiles(loadpath)

	currentSvcAccs, err := client.GetServiceAccounts()

	if err != nil {
		return err
	}
	for _, file := range files {

		var svcaccs []api.ServiceAccount
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &svcaccs); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}
		for _, svcacc := range svcaccs {
			found := false
			for _, csvcacc := range currentSvcAccs {
				if csvcacc.Name == svcacc.Name {
					found = true
				}
			}
			if found {
				continue
			}
			if err := client.CreateServiceAccount(&svcacc); err != nil {
				golog.Errorf("Error creating service account [%s] from [%s] [%s]", svcacc.Name, loadpath, err.Error())
				return err
			}
		}
	}
	golog.Infof("Created/updated service accounts from [%s]", loadpath)

	return nil
}
