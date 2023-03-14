package license

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/spf13/cobra"
)

// NewLicenseGroupCommand creates the `license` command
func NewLicenseGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "license",
		Short: "View or update Lenses license",
		Example: `lenses-cli license get
lenses-cli license update --license-file <license.json>`,
	}

	cmd.AddCommand(NewLicenseGetCommand())
	cmd.AddCommand(NewLicenseUpdateCommand())
	return cmd
}

// NewLicenseGetCommand creates the `license get` subcommand
func NewLicenseGetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Print information about the active Lenses license",
		Example: `lenses-cli license get`,
		RunE: func(cmd *cobra.Command, args []string) error {
			lc, err := config.Client.GetLicenseInfo()
			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, lc)
		},
	}

	bite.CanPrintJSON(cmd)

	return cmd
}

// NewLicenseUpdateCommand creates the `license update` subcommand
func NewLicenseUpdateCommand() *cobra.Command {
	var licenseFilePath string

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Update Lenses license by passing a JSON file",
		Example: `lenses-cli license update --file my-license.json`,
		RunE: func(cmd *cobra.Command, args []string) error {

			license, err := LoadLicenseFile(licenseFilePath)
			if err != nil {
				return err
			}

			err = config.Client.UpdateLicense(license)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "License updated")
			return nil
		},
	}

	cmd.Flags().StringVar(&licenseFilePath, "file", "", "The file path of the license file")
	cmd.MarkFlagRequired("file")
	return cmd
}

// LoadLicenseFile loads a file from filesystem and pass it for parsing
func LoadLicenseFile(licenseFilePath string) (api.License, error) {

	licenseFile, err := os.Open(licenseFilePath)
	defer licenseFile.Close()
	if err != nil {
		golog.Errorf("Failed to load license file", err.Error())
		return api.License{}, err
	}
	return ParseLicenseFile(licenseFile)
}

// ParseLicenseFile unmarshalls the license file into a known struct
func ParseLicenseFile(licenseFile io.Reader) (api.License, error) {
	var license api.License
	licenseFileAsBytes, _ := ioutil.ReadAll(licenseFile)
	err := json.Unmarshal(licenseFileAsBytes, &license)
	if err != nil {
		invalidLicenseErr := errors.New("invalid Lenses license JSON file")
		golog.Errorf(invalidLicenseErr.Error(), err.Error())
		return license, invalidLicenseErr
	}

	if (license == api.License{}) {
		emptyLicenseErr := errors.New("empty Lenses license file")
		return license, emptyLicenseErr
	}
	return license, nil
}
