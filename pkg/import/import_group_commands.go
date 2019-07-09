package imports

import (
	"github.com/landoop/bite"
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
import processors  --landscape my-acls-dir
import quota --landscape my-acls-dir
import schemas --landscape my-acls-dir
import topics --landscape my-acls-dir
import policies --landscape my-acls-dir`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.AddCommand(NewImportAclsCommand())
	cmd.AddCommand(NewImportAlertSettingsCommand())
	cmd.AddCommand(NewImportConnectorsCommand())
	cmd.AddCommand(NewImportProcessorsCommand())
	cmd.AddCommand(NewImportQuotasCommand())
	cmd.AddCommand(NewImportSchemasCommand())
	cmd.AddCommand(NewImportTopicsCommand())
	cmd.AddCommand(NewImportPoliciesCommand())

	return cmd
}

func load(cmd *cobra.Command, path string, data interface{}) error {
	return bite.TryReadFile(path, data)
}
