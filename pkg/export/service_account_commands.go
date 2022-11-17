package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportServiceAccountsCommand creates `export serviceaccounts`
func NewExportServiceAccountsCommand() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:              "serviceaccounts",
		Short:            "export serviceaccounts",
		Example:          `export serviceaccounts`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			if err := writeServiceAccounts(cmd, name); err != nil {
				golog.Errorf("Error writing service accounts. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().StringVar(&name, "name", "", "The service account name to extract")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeServiceAccounts(cmd *cobra.Command, accountName string) error {

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	if accountName != "" {
		svcAcc, err := config.Client.GetServiceAccount(accountName)
		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("svc-accounts-%s.%s", strings.ToLower(svcAcc.Name), strings.ToLower(output))
		return utils.WriteFile(landscapeDir, pkg.ServiceAccountsPath, fileName, output, svcAcc)
	}
	svcaccs, err := config.Client.GetServiceAccounts()
	if err != nil {
		return err
	}

	for _, svcAcc := range svcaccs {
		fileName := fmt.Sprintf("svc-accounts-%s.%s", strings.ToLower(svcAcc.Name), strings.ToLower(output))
		if accountName != "" && svcAcc.Name == accountName {
			return utils.WriteFile(landscapeDir, pkg.ServiceAccountsPath, fileName, output, svcAcc)
		}

		err := utils.WriteFile(landscapeDir, pkg.ServiceAccountsPath, fileName, output, svcAcc)
		if err != nil {
			return err
		}
	}
	return nil
}
