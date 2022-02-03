package provision

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	jsoniter "github.com/json-iterator/go"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/mitchellh/mapstructure"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
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
			setupCompleted, err := isSetupCompleted()
			if err != nil {
				return err
			}

			if !(setupModeFlag && setupCompleted) {
				if err := provision(cmd, args); err != nil {
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

func isSetupCompleted() (bool, error) {
	setupPayload := struct {
		IsCompleted bool `json:"isCompleted"`
	}{}

	resp, err := config.Client.Do(http.MethodGet, pkg.SetupPath, "application/json", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if err := config.Client.ReadJSON(resp, &setupPayload); err != nil {
		return false, err
	}

	return setupPayload.IsCompleted, nil
}

// provision func embeds the flow of provisioning logic
func provision(cmd *cobra.Command, args []string) error {
	yamlFileAsBytes, err := ioutil.ReadFile(args[0])
	if err != nil {
		return err
	}

	data := make(map[interface{}]interface{})
	if err := yaml.Unmarshal(yamlFileAsBytes, &data); err != nil {
		return err
	}

	// Check if input file has the expected structure
	if err := checkConfigValidity(data); err != nil {
		return err
	}

	// Parse config and replace 'fileRef' with 'fileId' only for connections
	if err := parseConfig(data, config.Client); err != nil {
		return err
	}

	// Decode generic config to high level known config struct
	var conf Config
	if err := mapstructure.Decode(data, &conf); err != nil {
		return err
	}

	// Handle connections
	for connName, conn := range conf.Connections {
		// Opted to use a 3rd party library since the standard one
		// cannot marshall a map of type map[interface{}]interface{}, only
		// map[string]interface{}
		jsoniter := jsoniter.ConfigCompatibleWithStandardLibrary
		jsonPayload, err := jsoniter.Marshal(&conn)
		if err != nil {
			return err
		}

		path := fmt.Sprintf("api/%s/%s", pkg.ConnectionsAPIPath, connName)
		resp, err := config.Client.Do(http.MethodPut, path, "application/json", jsonPayload)

		if err != nil {
			return err
		}

		defer resp.Body.Close()

		respAsBytes, err := config.Client.ReadResponseBody(resp)
		if err != nil {
			return err
		}

		// Let's grab the connection name for logging purposes
		connectionResponse := struct {
			Name string `json:"name"`
		}{}

		if err := jsoniter.Unmarshal(respAsBytes, &connectionResponse); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "connection '%s' configured successfully\n", connectionResponse.Name)

	}

	// License is optional value
	if conf.License.FileRef != (FileRef{}) {
		licenseAsBytes, _, err := fileRefToBytes(conf.License.FileRef)
		if err != nil {
			return err
		}

		var lic api.License
		if err := json.Unmarshal(licenseAsBytes, &lic); err != nil {
			return err
		}

		if err := config.Client.UpdateLicense(lic); err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "license updated successfully")
	}
	return nil
}
