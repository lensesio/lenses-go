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

var (
	testCurrentContextField                     = "master"
	testHostField                               = "https://landoop.com"
	testUsernameField                           = "testuser"
	testPasswordField                           = "testpassword"
	testBasicAuthenticationField                = BasicAuthentication{Username: testUsernameField, Password: testPasswordField}
	testKerberosConfFileField                   = "/etc/krb5.conf"
	testKerberosRealmField                      = "my.default"
	testKerberosAuthenticationWithPasswordField = KerberosAuthentication{
		ConfFile: testKerberosConfFileField,
		Method: KerberosWithPassword{
			Realm:    testKerberosRealmField,
			Username: testUsernameField,
			Password: testPasswordField,
		},
	}
	testTimeoutField  = "11s"
	testInsecureField = true
	testDebugField    = true
)

var (
	expectedConfigurationBasicAuthentication = Configuration{
		CurrentContext: testCurrentContextField,
		Contexts: map[string]*ClientConfiguration{
			testCurrentContextField: {
				Host:           testHostField,
				Authentication: testBasicAuthenticationField,
				Timeout:        testTimeoutField,
				Debug:          testDebugField,
			},
		},
	}

	expectedConfigurationKerberosAuthenticationWithPassword = Configuration{
		CurrentContext: testCurrentContextField,
		Contexts: map[string]*ClientConfiguration{
			testCurrentContextField: {
				Host:           testHostField,
				Authentication: testKerberosAuthenticationWithPasswordField,
				Timeout:        testTimeoutField,
				Debug:          testDebugField,
			},
		},
	}
)

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

func testConfigurationFile(t *testing.T, filename, contents string, reader func(string, *Configuration) error, expected Configuration) {
	f, teardown := makeTestFile(t, filename)
	defer teardown()

	f.WriteString(contents)

	var got Configuration
	if err := reader(f.Name(), &got); err != nil {
		t.Fatalf("[%s] while reading contents: '%s': %v,", t.Name(), contents, err)
	}

	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("[%s] error reading configuration from file: '%s'\nexpected:\n%#v\nbut got:\n%#v", t.Name(), f.Name(), expected, got)
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
				"%s": {
					"host": "%s",
					"basic_authentication": {"username": "%s", "password": "%s"},
					"timeout": "%s",
					"debug": %v
				}
			}
		}`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testUsernameField,
		testPasswordField,
		testTimeoutField,
		testDebugField)
	testConfigurationFile(t, "configuration.json", contents, ReadConfigurationFromJSON, expectedConfigurationBasicAuthentication)
}

func TestReadConfigurationFromJSONBackwardsCompatibility(t *testing.T) {
	contents := fmt.Sprintf(`
        {
			"currentContext": "%s",
			"contexts": {
				"%s": {
					"host": "%s",
					"user": "%s",
					"password": "%s",
					"timeout": "%s",
					"debug": %v
				}
			}
		}`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testUsernameField,
		testPasswordField,
		testTimeoutField,
		testDebugField)
	testConfigurationFile(t, "configuration.json", contents, ReadConfigurationFromJSON, expectedConfigurationBasicAuthentication)
}

func TestWriteConfigurationToJSON(t *testing.T) {
	expectedContents := fmt.Sprintf(`{"currentContext":"%s","contexts":{"%s":{"host":"%s","basic_authentication":{"username":"%s","password":"%s"},"timeout":"%s","debug":%v}}}`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testUsernameField,
		testPasswordField,
		testTimeoutField,
		testDebugField)

	b, err := ConfigurationMarshalJSON(expectedConfigurationBasicAuthentication)
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
    %s:
        Host: %s
        BasicAuthentication:
            Username: "%s"
            Password: "%s"
        Timeout: %s
        Debug: %v
`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testUsernameField,
		testPasswordField,
		testTimeoutField,
		testDebugField)
	testConfigurationFile(t, "configuration.yml", contents, ReadConfigurationFromYAML, expectedConfigurationBasicAuthentication)
}

func TestReadConfigurationFromYAMLBackwardsCompatibility(t *testing.T) {
	contents := fmt.Sprintf(`
        CurrentContext: %s
        Contexts:
            %s:
                Host: %s
                User: %s
                Password: %s
                Timeout: %s
                Debug: %v
        `,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testUsernameField,
		testPasswordField,
		testTimeoutField,
		testDebugField)
	testConfigurationFile(t, "configuration.yml", contents, ReadConfigurationFromYAML, expectedConfigurationBasicAuthentication)
}

func TestWriteConfigurationToYAML(t *testing.T) {
	expectedContents := fmt.Sprintf(`CurrentContext: %s
Contexts:
  %s:
    Host: %s
    Timeout: %s
    Debug: %v
    BasicAuthentication:
      Username: %s
      Password: %s`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testTimeoutField,
		testDebugField,
		testUsernameField,
		testPasswordField,
	)

	b, err := ConfigurationMarshalYAML(expectedConfigurationBasicAuthentication)
	if err != nil {
		t.Fatal(err)
	}

	// fmt.Println(strings.Replace(string(b), " ", "-", -1))

	if expected, got := strings.TrimSpace(expectedContents), strings.TrimSpace(string(b)); expected != got {
		t.Fatalf("expected result yaml to be written as:\n'%s'\nbut:\n'%s'", expected, got)
	}
}
