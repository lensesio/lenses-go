package imports

import (
	"github.com/lensesio/bite"
	"github.com/spf13/cobra"
)

//NewImportGroupCommand creates `import` command
func NewImportGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "import a landscape",
		Example: `
import acls --landscape my-acls-dir
import alert-settings --landscape my-acls-dir
import connectors --landscape my-acls-dir
import connections --landscape my-acls-dir
import processors  --landscape my-acls-dir
import quota --landscape my-acls-dir
import schemas --landscape my-acls-dir
import topics --landscape my-acls-dir
import policies --landscape my-acls-dir
import groups --dir groups
import serviceaccounts --dir serviceaccounts`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.AddCommand(NewImportAclsCommand())
	cmd.AddCommand(NewImportAlertSettingsCommand())
	cmd.AddCommand(NewImportConnectionsCommand())
	cmd.AddCommand(NewImportConnectorsCommand())
	cmd.AddCommand(NewImportProcessorsCommand())
	cmd.AddCommand(NewImportQuotasCommand())
	cmd.AddCommand(NewImportSchemasCommand())
	cmd.AddCommand(NewImportTopicsCommand())
	cmd.AddCommand(NewImportPoliciesCommand())
	cmd.AddCommand(NewImportGroupsCommand())
	cmd.AddCommand(NewImportServiceAccountsCommand())
	cmd.AddCommand(NewImportAlertChannelsCommand())
	cmd.AddCommand(NewImportAuditChannelsCommand())

	return cmd
}

func load(cmd *cobra.Command, path string, data interface{}) error {
	return bite.TryReadFile(path, data)
}
