package connection

import (
	"github.com/lensesio/lenses-go/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewWebhookGroupCommand(gen genericConnectionClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "webhook",
		Aliases:          []string{"wh"},
		Short:            "Manage Lenses Webhook connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[api.ConfigurationObjectWebhook]("Webhook", gen, up, FlagMapperOpts{
		Descriptions: map[string]string{
			"Host":     "The host name.",
			"UseHTTPs": "Set to true in order to set the URL scheme to `https`. Will otherwise default to `http`.",
			"Creds":    "An array of (secret) strings to be passed over to alert channel plugins.",
			"Port":     "An optional port number to be appended to the the hostname.",
		},
		Rename: map[string]string{
			"Host": "wh-host", // clashes.
		},
	})...)
	return cmd
}
