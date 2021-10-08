package beta

import (
	"encoding/json"
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

var configMode string

// NewRootCommand is the root command 'beta' used for experimental features
func NewRootCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "beta",
		Short:            "beta short desc",
		Long:             "beta long desc",
		TraverseChildren: true,
	}

	cmd.AddCommand(NewProvisionCommand())
	return cmd
}

// NewProvisionCommand is the 'beta provision' commmand
func NewProvisionCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:     "provision <config_yaml_file> [--mode {normal,sidecar}]",
		Long:    "Provision Lenses with a YAML config file to setup license, connections, etc",
		Example: "provision wizard.yml --mode=sidecar",
		RunE: func(cmd *cobra.Command, args []string) error {

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
			for _, conn := range conf.Connections {
				// Opted to use a 3rd party library since the standard one
				// cannot marshall a map of type map[interface{}]interface{}, only
				// map[string]interface{}
				jsoniter := jsoniter.ConfigCompatibleWithStandardLibrary
				jsonPayload, err := jsoniter.Marshal(&conn)
				if err != nil {
					return err
				}

				path := fmt.Sprintf("api/%s", pkg.ConnectionsAPIPath)
				resp, err := config.Client.Do(http.MethodPost, path, "application/JSON", jsonPayload)

				// Handle here if connection already exists ??
				if err != nil {
					fmt.Fprintln(cmd.ErrOrStderr(), err)
					continue
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
				fmt.Printf("connection '%s' configured successfully\n", connectionResponse.Name)

			}

			// Handle license
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

			// If --mode flag set to "sidecar" (for k8s purposes) then keep CLI running
			if configMode == "sidecar" {
				select {}
			}
			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&configMode, "mode", "normal", "[normal,sidecar]")
	return cmd
}
