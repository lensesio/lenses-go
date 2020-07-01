package license

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/test"
	"github.com/stretchr/testify/assert"
)

func TestLicenseGetCommand(t *testing.T) {
	testsLicenseGetCmd := []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"license get command",
			[]string{""},
			`{"clientId":"Studio Beta","isRespected":true,"maxBrokers":69,"expiry":1612137598000,"monthsToExpire":7}`,
			errors.New(""),
		},
	}

	sampleLicense := api.LicenseInfo{
		ClientID:    "Studio Beta",
		IsRespected: true,
		MaxBrokers:  69,
		MaxMessages: 0,
		Expiry:      1612137598000,
	}
	json, _ := json.Marshal(sampleLicense)
	for _, tt := range testsLicenseGetCmd {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(json))
		})
		httpClient, teardown := test.TestingHTTPClient(h)
		defer teardown()
		client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
		assert.Nil(t, err)
		config.Client = client

		t.Run(tt.name, func(t *testing.T) {
			cmd := NewLicenseGetCommand()
			var outputValue string
			cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
			out, _ := test.ExecuteCommand(cmd, tt.args...)

			test.CheckStringContains(t, out, tt.expectOut)
		})
	}
}

func TestLicenseUpdateCommand(t *testing.T) {

	testsLicenseUpdateCmd := []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"license group command",
			[]string{""},
			"View or update Lenses license",
			errors.New(""),
		},
		{
			"Missing --file flag",
			[]string{"update"},
			"Error: required flag(s) \"file\" not set",
			errors.New(""),
		},
		{
			"Inexistant license file",
			[]string{"update", "--file", "imaginary.file"},
			"open imaginary.file: no such file or directory",
			errors.New(""),
		},
		{
			"Invalid license file",
			[]string{"update", "--file", "commands.go"},
			"Invalid Lenses license JSON file",
			errors.New(""),
		},
	}

	for _, tt := range testsLicenseUpdateCmd {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(nil))
		})
		httpClient, teardown := test.TestingHTTPClient(h)
		defer teardown()
		client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
		assert.Nil(t, err)
		config.Client = client

		t.Run(tt.name, func(t *testing.T) {
			cmd := NewLicenseGroupCommand()
			out, _ := test.ExecuteCommand(cmd, tt.args...)

			test.CheckStringContains(t, out, tt.expectOut)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
				return
			}
			if err == nil && tt.expectError.Error() != "" {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
				return
			}
		})
	}
}

func TestParseLicenseFile(t *testing.T) {
	validLicenseFileContent := `{"source":"Lenses.io","clientId":"6969","details":"foobar","key":"1978"}`
	invalidFileType := "# This is a README file"
	invalidJSONLicenseContent := "{ \"foo\":\"bar\"}"

	testsLicenseFileContent := []struct {
		name        string
		fileContent io.Reader
		expectError error
	}{
		{
			"Valid license",
			strings.NewReader(validLicenseFileContent),
			errors.New(""),
		},
		{
			"Invalid file type (not a valid JSON)",
			strings.NewReader(invalidFileType),
			errors.New("Invalid Lenses license JSON file"),
		},
		{
			"Valid JSON but invalid license content",
			strings.NewReader(invalidJSONLicenseContent),
			errors.New("Empty Lenses license file"),
		},
	}

	for _, tt := range testsLicenseFileContent {
		t.Run(tt.name, func(t *testing.T) {
			_, err := parseLicenseFile(tt.fileContent)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
				return
			}
			if err == nil && tt.expectError.Error() != "" {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
				return
			}
		})
	}
}
