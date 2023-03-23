package connection

import (
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	cobra "github.com/spf13/cobra"
)

// NewConnectionGroupCommand creates `connection` command
func NewConnectionGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "connections",
		Short:            `Manage Lenses external connections`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.AddCommand(NewGenericConnectionGroupCommand())

	// Maintain the generic connection manipulation commands where they used to
	// be for compatibility, but hidden and marked as deprecated.
	cmd.AddCommand(hideAndDeprecate(`this command has been moved under the "generic" connections command`,
		NewGenericConnectionGetCommand(),
		NewGenericConnectionCreateCommand(),
		NewGenericConnectionDeleteCommand(),
		NewGenericConnectionUpdateCommand(),
		NewGenericConnectionListCommand(),
	)...)

	cmd.AddCommand(
		// Those commands use the specific endpoints.
		NewKafkaGroupCommand(config.Client, upload),
		NewKafkaConnectGroupCommand(upload),
		NewKerberosGroupCommand(upload),
		NewSchemaRegistryGroupCommand(upload),
		NewZookeeperGroupCommand(upload),
		// Those commands use the generic endpoints.
		NewAWSGroupCommand(config.Client, upload),
		NewDataDogGroupCommand(config.Client, upload),
		NewElasticsearchGroupCommand(config.Client, upload),
		NewPagerDutyGroupCommand(config.Client, upload),
		NewPostgreSQLGroupCommand(config.Client, upload),
		NewPrometheusAlertmanagerGroupCommand(config.Client, upload),
		NewSlackGroupCommand(config.Client, upload),
		NewSplunkGroupCommand(config.Client, upload),
		NewWebhookGroupCommand(config.Client, upload),
	)

	return cmd
}

func hideAndDeprecate(deprecate string, cs ...*cobra.Command) []*cobra.Command {
	for _, c := range cs {
		c.Hidden = true
		c.Deprecated = deprecate
	}
	return cs
}
