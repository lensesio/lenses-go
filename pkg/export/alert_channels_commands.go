package export

import (
	"fmt"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/spf13/cobra"
)

// NewExportAlertChannelsCommand creates `export alert-channels` command
func NewExportAlertChannelsCommand() *cobra.Command {
	var alertChannelName string

	cmd := &cobra.Command{
		Use:              "alert-channels",
		Short:            "export alert-channels",
		Example:          `export alert-channels --resource-name=my-alert`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			if err := writeChannels(cmd, pkg.AlertChannelsPath, "alert", alertChannelName); err != nil {
				return fmt.Errorf("failed to export alert channels from server: [%v]", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().StringVar(&alertChannelName, "resource-name", "", "The name of the alert channel to export")
	//cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract alert channel dependencies, e.g. connections")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}
