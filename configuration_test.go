// Black-box testing for the configuration readers.
package lenses

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

// TODO: will fail atm, will be fixed when CLI adapts the latest client's API changes.

const testDebug = false

var expectedConfiguration = Configuration{
	CurrentContext: "master",
	Contexts: map[string]*ClientConfiguration{
		"master": {
			Host:           "https://landoop.com",
			Authentication: BasicAuthentication{Username: "testuser", Password: "testpassword"},
			Timeout:        "11s",
			Debug:          true,
		},
	},
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

func testConfigurationFile(t *testing.T, filename, contents string, reader func(string, *Configuration) error) {
	f, teardown := makeTestFile(t, filename)
	defer teardown()

	f.WriteString(contents)

	var got Configuration
	if err := reader(f.Name(), &got); err != nil {
		t.Fatalf("[%s] while reading contents: '%s': %v,", t.Name(), contents, err)
	}

	if !reflect.DeepEqual(got, expectedConfiguration) {
		t.Fatalf("[%s] error reading configuration from file: '%s'\nexpected:\n%#v\nbut got:\n%#v", t.Name(), f.Name(), expectedConfiguration, got)
		if testDebug {
			t.Fatalf(" \ncontents of the file:\n%s", contents)
		}
	}
}

func TestReadConfigurationFromJSON(t *testing.T) {
	contents := fmt.Sprintf(`
        {
			"currentContext": "%s",
			"contexts": {
				"master": {
					"host": "%s",
					"basic_authentication": {"username": "%s", "password": "%s"},
					"timeout": "%s",
					"debug": %v
				}
			}
		}`,
		expectedConfiguration.CurrentContext,
		expectedConfiguration.GetCurrent().Host,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Username,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Password,
		expectedConfiguration.GetCurrent().Timeout,
		expectedConfiguration.GetCurrent().Debug)
	testConfigurationFile(t, "configuration.json", contents, ReadConfigurationFromJSON)
}

func TestReadConfigurationFromJSONBackwardsCompatibility(t *testing.T) {
	contents := fmt.Sprintf(`
        {
			"currentContext": "%s",
			"contexts": {
				"master": {
					"host": "%s",
					"user": "%s",
					"password": "%s",
					"timeout": "%s",
					"debug": %v
				}
			}
		}`,
		expectedConfiguration.CurrentContext,
		expectedConfiguration.GetCurrent().Host,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Username,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Password,
		expectedConfiguration.GetCurrent().Timeout,
		expectedConfiguration.GetCurrent().Debug)
	testConfigurationFile(t, "configuration.json", contents, ReadConfigurationFromJSON)
}

func TestWriteConfigurationToJSON(t *testing.T) {
	expectedContents := fmt.Sprintf(`{"currentContext":"%s","contexts":{"master":{"host":"%s","basic_authentication":{"username":"%s","password":"%s"},"timeout":"%s","debug":%v}}}`,
		expectedConfiguration.CurrentContext,
		expectedConfiguration.GetCurrent().Host,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Username,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Password,
		expectedConfiguration.GetCurrent().Timeout,
		expectedConfiguration.GetCurrent().Debug)

	b, err := ConfigurationMarshalJSON(expectedConfiguration)
	if err != nil {
		t.Fatal(err)
	}

	if expected, got := strings.TrimSpace(expectedContents), strings.TrimSpace(string(b)); expected != got {
		t.Fatalf("expected result json to be written as:\n'%s'\nbut:\n'%s'", expected, got)
	}
}

func TestReadConfigurationFromYAML(t *testing.T) {
	contents := fmt.Sprintf(`
CurrentContext: %s
Contexts:
    master:
        Host: %s
        BasicAuthentication:
            Username: "%s"
            Password: "%s"
        Timeout: %s
        Debug: %v
`,
		expectedConfiguration.CurrentContext,
		expectedConfiguration.GetCurrent().Host,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Username,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Password,
		expectedConfiguration.GetCurrent().Timeout,
		expectedConfiguration.GetCurrent().Debug)
	testConfigurationFile(t, "configuration.yml", contents, ReadConfigurationFromYAML)
}

func TestReadConfigurationFromYAMLBackwardsCompatibility(t *testing.T) {
	contents := fmt.Sprintf(`
        CurrentContext: %s
        Contexts:
            master:
                Host: %s
                User: %s
                Password: %s
                Timeout: %s
                Debug: %v
        `,
		expectedConfiguration.CurrentContext,
		expectedConfiguration.GetCurrent().Host,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Username,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Password,
		expectedConfiguration.GetCurrent().Timeout,
		expectedConfiguration.GetCurrent().Debug)
	testConfigurationFile(t, "configuration.yml", contents, ReadConfigurationFromYAML)
}

func TestWriteConfigurationToYAML(t *testing.T) {
	// 	expectedContents := fmt.Sprintf(`CurrentContext: %s
	// Contexts:
	//   master:
	//     Host: %s
	//     Token: ""
	//     Timeout: %s
	//     Debug: %v
	//     BasicAuthentication:
	//       Username: %s
	//       Password: %s
	//       `,
	expectedContents := fmt.Sprintf(`CurrentContext: %s
Contexts:
  master:
    Host: %s
    Token: ""
    Timeout: %s
    Debug: %v
    BasicAuthentication:
      Username: %s
      Password: %s`,
		expectedConfiguration.CurrentContext,
		expectedConfiguration.GetCurrent().Host,
		expectedConfiguration.GetCurrent().Timeout,
		expectedConfiguration.GetCurrent().Debug,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Username,
		expectedConfiguration.GetCurrent().Authentication.(BasicAuthentication).Password,
	)

	b, err := ConfigurationMarshalYAML(expectedConfiguration)
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(strings.Replace(string(b), " ", "-", -1))

	if expected, got := strings.TrimSpace(expectedContents), strings.TrimSpace(string(b)); expected != got {
		t.Fatalf("expected result yaml to be written as:\n'%s'\nbut:\n'%s'", expected, got)
	}
}
