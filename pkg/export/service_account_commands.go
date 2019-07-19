package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewExportServiceAccountsCommand creates `export serviceaccounts`
func NewExportServiceAccountsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "serviceaccounts",
		Short:            "export serviceaccounts",
		Example:          `export serviceaccounts`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			svcaccs, err := config.Client.GetServiceAccounts()
			if err != nil {
				golog.Errorf("Failed to find service accounts. [%s]", err.Error())
				return err
			}
			if err := writeServiceAccounts(cmd, &svcaccs); err != nil {
				golog.Errorf("Error writing service accounts. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeServiceAccounts(cmd *cobra.Command, svcaccs *[]api.ServiceAccount) error {

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("service-accounts.%s", strings.ToLower(output))
	return utils.WriteFile(landscapeDir, pkg.ServiceAccountsPath, fileName, output, svcaccs)
}
