package export

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
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

	// Since service accounts name can differ in case, we use a hash to ensure uniqueness
	// and avoid writing one file over another with the same name
	h := fnv.New64a()
	for _, svcAcc := range svcaccs {
		h.Write([]byte(svcAcc.Name))

		lowerSvcAccName := strings.ToLower(svcAcc.Name)
		fileName := fmt.Sprintf("svc-accounts-%s-%08x.%s", lowerSvcAccName, uint32(h.Sum64()), strings.ToLower(output))

		if accountName != "" && svcAcc.Name == accountName {
			return utils.WriteFile(landscapeDir, pkg.ServiceAccountsPath, fileName, output, svcAcc)
		}

		err = utils.WriteFile(landscapeDir, pkg.ServiceAccountsPath, fileName, output, svcAcc)
		if err != nil {
			return err
		}
	}
	return nil
}
