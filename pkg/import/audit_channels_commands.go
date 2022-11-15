package imports

import (
	"fmt"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

// NewImportAuditChannelsCommand handles the CLI sub-command 'import audit-channels'
func NewImportAuditChannelsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "audit-channels",
		Short:            "audit-channels",
		Example:          `import audit-channels --dir <dir>`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, "audit-channels")

			err := importChannels(config.Client, cmd, path, "audit", pkg.AuditChannelsPath)
			if err != nil {
				return fmt.Errorf("error importing audit channels. [%v]", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}
