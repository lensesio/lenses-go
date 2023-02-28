package connection

import (
	"github.com/lensesio/lenses-go/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewDataDogGroupCommand(gen genericConnectionClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "data-dog",
		Aliases:          []string{"dd"},
		Short:            "Manage Lenses DataDog connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[api.ConfigurationObjectDataDog]("DataDog", gen, up, FlagMapperOpts{
		Descriptions: map[string]string{
			"APIKey":         "The Datadog API key.",
			"Site":           "The Datadog site, e.g. EU or US.",
			"ApplicationKey": "The Datadog application key.",
		},
	})...)
	return cmd
}
