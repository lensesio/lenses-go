package imports

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	quotapkg "github.com/lensesio/lenses-go/pkg/quota"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewImportQuotasCommand creates `import quotas` command
func NewImportQuotasCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "quotas",
		Short:            "quotas",
		Example:          `import quotas --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.QuotasPath)
			if err := loadQuotas(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to load quotas. [%s]", err.Error())
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

func loadQuotas(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading quotas from [%s]", loadpath)
	files, err := utils.FindFiles(loadpath)
	if err != nil {
		return err
	}

	lensesQuotas, err := client.GetQuotas()
	var lensesReq []api.CreateQuotaPayload

	if err != nil {
		return err
	}

	for _, lq := range lensesQuotas {
		lensesReq = append(lensesReq, lq.GetQuotaAsRequest())
	}

	for _, file := range files {
		var quotas []api.CreateQuotaPayload
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &quotas); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		for _, quota := range quotas {

			found := false
			for _, lq := range lensesReq {
				if quota.ClientID == lq.ClientID &&
					quota.QuotaType == lq.QuotaType &&
					quota.User == lq.User &&
					quota.Config.ConsumerByteRate == lq.Config.ConsumerByteRate &&
					quota.Config.ProducerByteRate == lq.Config.ProducerByteRate &&
					quota.Config.RequestPercentage == lq.Config.RequestPercentage {
					found = true
				}
			}

			if found {
				continue
			}

			if quota.QuotaType == string(api.QuotaEntityClient) ||
				quota.QuotaType == string(api.QuotaEntityClients) ||
				quota.QuotaType == string(api.QuotaEntityClientsDefault) {
				if err := quotapkg.CreateQuotaForClients(cmd, client, quota); err != nil {
					golog.Errorf("Error creating/updating quota type [%s], client [%s], user [%s] from [%s]. [%s]",
						quota.QuotaType, quota.ClientID, quota.User, loadpath, err.Error())
					return err
				}

				golog.Infof("Created/updated quota type [%s], client [%s], user [%s] from [%s]",
					quota.QuotaType, quota.ClientID, quota.User, loadpath)
				continue

			}

			if err := quotapkg.CreateQuotaForUsers(cmd, client, quota); err != nil {
				golog.Errorf("Error creating/updating quota type [%s], client [%s], user [%s] from [%s]. [%s]",
					quota.QuotaType, quota.ClientID, quota.User, loadpath, err.Error())
				return err
			}

			golog.Infof("Created/updated quota type [%s], client [%s], user [%s] from [%s]",
				quota.QuotaType, quota.ClientID, quota.User, loadpath)
		}
	}
	return nil
}
