package connection

import (
	"github.com/lensesio/lenses-go/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewElasticsearchGroupCommand(gen genericConnectionClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "elasticsearch",
		Aliases:          []string{"elastic", "es"},
		Short:            "Manage Lenses Elasticsearch connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[api.ConfigurationObjectElasticsearch]("Elasticsearch", gen, up, FlagMapperOpts{
		Rename: map[string]string{
			"User":     "es-user",     // clashes with persistent flag.
			"Password": "es-password", // for consistency.
		},
		Descriptions: map[string]string{
			"Nodes":    "The nodes of the Elasticsearch cluster to connect to, e.g. https://hostname:port.",
			"Password": "The password to connect to the Elasticsearch service.",
			"User":     "The username to connect to the Elasticsearch service.",
		},
	})...)
	return cmd
}
