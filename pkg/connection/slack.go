package connection

import (
	"github.com/lensesio/lenses-go/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewSlackGroupCommand(gen genericConnectionClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "slack",
		Short:            "Manage Lenses Slack connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[api.ConfigurationObjectSlack]("Slack", gen, up, FlagMapperOpts{
		Descriptions: map[string]string{
			"WebhookURL": "The Slack endpoint to send the alert to.",
		},
	})...)
	return cmd
}
