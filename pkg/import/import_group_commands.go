package imports

import (
	"github.com/lensesio/bite"
	"github.com/spf13/cobra"
)

// NewImportGroupCommand creates `import` command
func NewImportGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "import",
		Short: "import a landscape",
		Example: `
import acls --dir my-acls-dir
import alert-settings --dir my-acls-dir
import connectors --dir my-acls-dir
import connections --dir my-acls-dir
import processors  --dir my-acls-dir
import quota --dir my-acls-dir
import schemas --dir my-acls-dir
import topics --dir my-acls-dir
import policies --dir my-acls-dir
import groups --dir groups
import topic-settings --dir topic-settings
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
	cmd.AddCommand(NewImportTopicsCommand())
	cmd.AddCommand(NewImportPoliciesCommand())
	cmd.AddCommand(NewImportGroupsCommand())
	cmd.AddCommand(NewImportServiceAccountsCommand())
	cmd.AddCommand(NewImportAlertChannelsCommand())
	cmd.AddCommand(ImportTopicSettingsCmd())
	cmd.AddCommand(NewImportAuditChannelsCommand())
	cmd.AddCommand(NewImportSchemasCmd())

	return cmd
}

func load(cmd *cobra.Command, path string, data interface{}) error {
	return bite.TryReadFile(path, data)
}
