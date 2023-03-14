package connection

import (
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	cobra "github.com/spf13/cobra"
)

func NewKerberosGroupCommand(up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "kerberos",
		Aliases:          []string{"kerb"},
		Short:            "Manage Lenses Kerberos connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	var testReq api.KerberosConnectionTestRequest
	var upsertReq api.KerberosConnectionUpsertRequest
	cmd.AddCommand(connCrudsToCobra("Kerberos", up,
		connCrud{
			use:           "get",
			runWithArgRet: func(arg string) (interface{}, error) { return config.Client.GetKerberosConnection(arg) },
		}, connCrud{
			use:         "list",
			runNoArgRet: func() (interface{}, error) { return config.Client.ListKerberosConnections() },
		}, connCrud{
			use:        "test",
			defaultArg: "kerberos",
			opts: FlagMapperOpts{
				Descriptions: kerbdescs,
				Hide:         []string{"Name"},
			},
			onto: &testReq,
			runWithargNoRet: func(arg string) error {
				testReq.Name = arg
				return config.Client.TestKerberosConnection(testReq)
			},
		}, connCrud{
			use:        "upsert",
			defaultArg: "kerberos",
			opts: FlagMapperOpts{
				Descriptions: kerbdescs,
			},
			onto: &upsertReq,
			runWithArgRet: func(arg string) (interface{}, error) {
				return config.Client.UpdateKerberosConnection(arg, upsertReq)
			},
		}, connCrud{
			use:             "delete",
			runWithargNoRet: config.Client.DeleteKerberosConnection,
		})...)

	return cmd
}

var kerbdescs = map[string]string{
	"KerberosKrb5": "Kerberos krb5.conf file.",
	"Update":       "Set to true if testing an update to an existing connection, false if testing a new connection.",
	"Tags":         "Any tags to add to the connection's metadata.",
}
