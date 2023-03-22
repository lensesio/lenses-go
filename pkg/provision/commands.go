package provision

import (
	"errors"
	"fmt"
	"net/http"
	"os"

	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/spf13/cobra"
)

var (
	configMode    string
	setupModeFlag bool
)

// NewProvisionCommand is the 'beta provision' commmand
func NewProvisionCommand() *cobra.Command {

	cmdShortDesc := "Provision Lenses with a YAML config file to setup license, connections, etc."
	cmdLongDesc := `Provision Lenses with a YAML config file to setup license, connections, etc..
If --mode flag set to 'sidecar' (for k8s purposes) then keep CLI running.`

	cmd := &cobra.Command{
		Use:     "provision <config_yaml_file> [--mode {normal,sidecar}] [--setup-mode]",
		Short:   cmdShortDesc,
		Long:    cmdLongDesc,
		Example: "provision wizard.yml --mode=sidecar",
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(args) == 0 {
				return errors.New("missing provisioning file, refer to `provision --help` for more info")
			}

			// If '--setup-mode' flag is set and setup has completed then skip provisioning
			status, err := config.Client.GetSetupStatus()
			if err != nil {
				return err
			}

			if !(setupModeFlag && status.IsCompleted) {
				yamlFileAsBytes, err := os.ReadFile(args[0])
				if err != nil {
					return err
				}
				if err := provision(yamlFileAsBytes, config.Client, http.DefaultClient); err != nil {
					return err
				}
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "skipping provisioning as Lenses setup has already been completed")
			}

			if configMode == "sidecar" {
				fmt.Fprintln(cmd.OutOrStdout(), "lenses-cli will stay in idle state")
				// An empty select block will keep the go processes running
				select {}
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&configMode, "mode", "normal", "[normal,sidecar]")
	cmd.PersistentFlags().BoolVar(&setupModeFlag, "setup-mode", false, "When set will perform the provision only if Lenses is still in setup/wizard mode")
	return cmd
}
