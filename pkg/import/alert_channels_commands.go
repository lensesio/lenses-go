package imports

import (
	"fmt"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/spf13/cobra"
)

// NewImportAlertChannelsCommand handles the CLI sub-command 'import alert-channels'
func NewImportAlertChannelsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "alert-channels",
		Short:            "alert-channels",
		Example:          `import alert-channels --dir <dir>`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, "alert-channels")

			err := importChannels(config.Client, cmd, path, "alert", pkg.AlertChannelsPath)
			if err != nil {
				return fmt.Errorf("error importing alert channels. [%v]", err)
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
