package connection

import (
	"github.com/lensesio/lenses-go/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewPagerDutyGroupCommand(gen genericConnectionClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "pager-duty",
		Aliases:          []string{"pd"},
		Short:            "Manage Lenses PagerDuty connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[api.ConfigurationObjectPagerDuty]("PagerDuty", gen, up, FlagMapperOpts{
		Descriptions: map[string]string{
			"IntegrationKey": "An Integration Key for PagerDuty's service with Events API v2 integration type.",
		},
	})...)
	return cmd
}
