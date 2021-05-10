package export

import (
	"fmt"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/spf13/cobra"
)

//NewExportAuditChannelsCommand creates `export audit-channels` command
func NewExportAuditChannelsCommand() *cobra.Command {
	var auditChannelName string

	cmd := &cobra.Command{
		Use:              "audit-channels",
		Short:            "export audit-channels",
		Example:          `export audit-channels --resource-name=my-audit`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			if err := writeChannels(cmd, pkg.AuditChannelsPath, "audit", auditChannelName); err != nil {
				return fmt.Errorf("failed to export audit channels from server: [%v]", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().StringVar(&auditChannelName, "resource-name", "", "The name of the audit channel to export")
	//cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract audit channel dependencies, e.g. connections")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}
