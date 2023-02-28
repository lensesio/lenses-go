package connection

import (
	"github.com/lensesio/lenses-go/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewPrometheusAlertmanagerGroupCommand(gen genericConnectionClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "prometheus",
		Aliases:          []string{"prom", "alert-manager"},
		Short:            "Manage Lenses PrometheusAlertmanager connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[api.ConfigurationObjectPrometheusAlertmanager]("PrometheusAlertmanager", gen, up, FlagMapperOpts{
		Descriptions: map[string]string{
			"Endpoints": "List of Alert Manager endpoints.",
		},
	})...)
	return cmd
}
