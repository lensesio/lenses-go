package connection

import (
	"github.com/lensesio/lenses-go/v5/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewSplunkGroupCommand(gen genericConnectionClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "splunk",
		Short:            "Manage Lenses Splunk connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[api.ConfigurationObjectSplunk]("Splunk", gen, up, FlagMapperOpts{
		Descriptions: map[string]string{
			"Host":     "The host name for the HTTP Event Collector API of the Splunk instance.",
			"Insecure": "This is not encouraged but is required for a Splunk Cloud Trial instance.",
			"Token":    "HTTP event collector authorization token.",
			"UseHTTPs": "Use SSL.",
			"Port":     "The port number for the HTTP Event Collector API of the Splunk instance.",
		},
		Rename: map[string]string{
			"Host":     "pg-host",     // clashes.
			"Insecure": "pg-insecure", // clashes.
			"Token":    "pg-token",    // clashes.
		},
	})...)
	return cmd
}
