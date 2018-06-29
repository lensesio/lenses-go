// Black-box testing for the configuration readers.
package lenses_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/landoop/lenses-go"
)

// TODO: will fail atm, will be fixed when CLI adapts the latest client's API changes.

const testDebug = false

var expectedConfiguration = lenses.Configuration{
	Host:           "https://landoop.com",
	Authentication: lenses.BasicAuthentication{Username: "testuser", Password: "testpassword"},
	Timeout:        "11s",
	Debug:          true,
}

func makeTestFile(t *testing.T, filename string) (*os.File, func()) {
	f, err := ioutil.TempFile("", filename)
	if err != nil {
		t.Fatalf("error creating the temp file: %v", err)
	}

	teardown := func() {
		f.Close()
		os.Remove(f.Name())
	}

	return f, teardown
}

func testConfigurationFile(t *testing.T, filename, contents string, reader func(string, *lenses.Configuration) error) {
	f, teardown := makeTestFile(t, filename)
	defer teardown()

	f.WriteString(contents)

	var got lenses.Configuration
	if err := reader(f.Name(), &got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, expectedConfiguration) {
		t.Fatalf("error reading configuration from file: '%s'\nexpected:\n%#v\nbut got:\n%#v", f.Name(), expectedConfiguration, got)
		if testDebug {
			t.Fatalf("\ncontents of the file:\n%s", contents)
		}
	}
}

func TestReadConfigurationFromJSON(t *testing.T) {
	contents := fmt.Sprintf(`
        {
            "host": "%s",
			"basic_authentication": {"username": "%s", "password": "%s"},
            "timeout": "%s",
            "debug": %v
        }`,
		expectedConfiguration.Host,
		expectedConfiguration.Authentication.(lenses.BasicAuthentication).Username,
		expectedConfiguration.Authentication.(lenses.BasicAuthentication).Password,
		expectedConfiguration.Timeout,
		expectedConfiguration.Debug)
	testConfigurationFile(t, "configuration.json", contents, lenses.ReadConfigurationFromJSON)
}

func TestReadConfigurationFromJSONBackwardsCompatibility(t *testing.T) {
	contents := fmt.Sprintf(`
        {
            "host": "%s",
			"user": "%s",
            "password": "%s",
            "timeout": "%s",
            "debug": %v
        }`,
		expectedConfiguration.Host,
		expectedConfiguration.Authentication.(lenses.BasicAuthentication).Username,
		expectedConfiguration.Authentication.(lenses.BasicAuthentication).Password,
		expectedConfiguration.Timeout,
		expectedConfiguration.Debug)
	testConfigurationFile(t, "configuration.json", contents, lenses.ReadConfigurationFromJSON)
}

func TestWriteConfigurationToJSON(t *testing.T) {
	expectedContents := fmt.Sprintf(`{"host":"%s","basic_authentication":{"username":"%s","password":"%s"},"timeout":"%s","debug":%v}`,
		expectedConfiguration.Host,
		expectedConfiguration.Authentication.(lenses.BasicAuthentication).Username,
		expectedConfiguration.Authentication.(lenses.BasicAuthentication).Password,
		expectedConfiguration.Timeout,
		expectedConfiguration.Debug)

	b, err := lenses.ConfigurationJSONMarshal(expectedConfiguration)
	if err != nil {
		t.Fatal(err)
	}

	if expected, got := strings.TrimSpace(expectedContents), strings.TrimSpace(string(b)); expected != got {
		t.Fatalf("expected result json to be written as:\n'%s'\nbut:\n'%s'", expected, got)
	}
}

func TestReadConfigurationFromYAML(t *testing.T) {
	contents := fmt.Sprintf(`
Host: %s
BasicAuthentication:
  Username: "%s"
  Password: "%s"
Timeout: %s
Debug: %v
        `,
		expectedConfiguration.Host,
		expectedConfiguration.Authentication.(lenses.BasicAuthentication).Username,
		expectedConfiguration.Authentication.(lenses.BasicAuthentication).Password,
		expectedConfiguration.Timeout,
		expectedConfiguration.Debug)
	testConfigurationFile(t, "configuration.yml", contents, lenses.ReadConfigurationFromYAML)
}
