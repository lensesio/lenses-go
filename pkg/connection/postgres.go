package connection

import (
	"github.com/lensesio/lenses-go/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewPostgreSQLGroupCommand(gen genericConnectionClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "postgresql",
		Aliases:          []string{"pg"},
		Short:            "Manage Lenses PostgreSQL connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[api.ConfigurationObjectPostgreSQL]("PostgreSQL", gen, up, FlagMapperOpts{
		Descriptions: map[string]string{
			"Database": "The database to connect to.",
			"Host":     "The Postgres hostname.",
			"Port":     "The port number.",
			"SslMode":  "The SSL connection mode as detailed in https://jdbc.postgresql.org/documentation/head/ssl-client.html.",
			"Username": "The user name.",
			"Password": "The password.",
		},
		Rename: map[string]string{
			"Host": "pg-host", // clashes.
		},
	})...)
	return cmd
}
