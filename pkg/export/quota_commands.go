package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewExportQuotasCommand creates `export quotas` command
func NewExportQuotasCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "quotas",
		Short:            "export quotas",
		Example:          `export quoats`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			if err := writeQuotas(cmd, config.Client); err != nil {
				golog.Errorf("Error writing quotas. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract dependencies, topics, acls, quotas, alerts")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func writeQuotas(cmd *cobra.Command, client *api.Client) error {

	quotas, err := client.GetQuotas()

	if err != nil {
		return err
	}

	var requests []api.CreateQuotaPayload
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("quotas.%s", strings.ToLower(output))

	for _, q := range quotas {
		requests = append(requests, q.GetQuotaAsRequest())
	}

	return utils.WriteFile(landscapeDir, pkg.QuotasPath, fileName, output, requests)
}
