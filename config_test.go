package lenses

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"testing"
)

const testDebug = false

var (
	testCurrentContextField             = "master"
	testHostField                       = "https://landoop.com"
	testUsernameField                   = "testuser"
	testPasswordField                   = "testpassword"
	testBasicAuthenticationField        = BasicAuthentication{Username: testUsernameField, Password: testPasswordField}
	testKerberosConfFileField           = "/etc/krb5.conf"
	testKerberosRealmField              = "my.default"
	testKerberosKeytabField             = "/tmp/my.keytab"
	testKerberosCCacheField             = "/tmp/ccache.file"
	testKerberosMethodWithPasswordField = KerberosWithPassword{
		Realm:    testKerberosRealmField,
		Username: testUsernameField,
		Password: testPasswordField,
	}
	testKerberosMethodWithKeytabField = KerberosWithKeytab{
		Username:   testUsernameField,
		Realm:      testKerberosRealmField,
		KeytabFile: testKerberosKeytabField,
	}
	testKerberosMethodFromCCacheField = KerberosFromCCache{CCacheFile: testKerberosCCacheField}
	testTimeoutField                  = "11s"
	testInsecureField                 = true
	testDebugField                    = true
)

var (
	expectedConfigurationBasicAuthentication = Config{
		CurrentContext: testCurrentContextField,
		Contexts: map[string]*ClientConfig{
			testCurrentContextField: {
				Host:           testHostField,
				Authentication: testBasicAuthenticationField,
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

func testConfigurationFile(t *testing.T, filename, contents string, reader func(string, *Config) error, expected Config) {
	f, teardown := makeTestFile(t, filename)
	defer teardown()

	f.WriteString(contents)

	var got Config
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

func TestReadConfigFromJSON(t *testing.T) {
	contents := fmt.Sprintf(`
        {
			"currentContext": "%s",
			"contexts": {
				"%s": {
					"host": "%s",
					"%s": {"username": "%s", "password": "%s"},
					"timeout": "%s",
					"debug": %v
				}
			}
		}`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		basicAuthenticationKeyJSON,
		testUsernameField,
		testPasswordField,
		testTimeoutField,
		testDebugField)
	testConfigurationFile(t, "configuration.json", contents, ReadConfigFromJSON, expectedConfigurationBasicAuthentication)
}

func TestReadConfigFromJSONBackwardsCompatibility(t *testing.T) {
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
	testConfigurationFile(t, "configuration.json", contents, ReadConfigFromJSON, expectedConfigurationBasicAuthentication)
}

func TestWriteConfigToJSON(t *testing.T) {
	expectedContents := fmt.Sprintf(`{"currentContext":"%s","contexts":{"%s":{"host":"%s","%s":{"username":"%s","password":"%s"},"timeout":"%s","debug":%v}}}`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		basicAuthenticationKeyJSON,
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

func TestReadConfigFromYAML(t *testing.T) {
	contents := fmt.Sprintf(`
CurrentContext: %s
Contexts:
    %s:
        Host: %s
        %s:
            Username: "%s"
            Password: "%s"
        Timeout: %s
        Debug: %v
`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		basicAuthenticationKeyYAML,
		testUsernameField,
		testPasswordField,
		testTimeoutField,
		testDebugField)
	testConfigurationFile(t, "configuration.yml", contents, ReadConfigFromYAML, expectedConfigurationBasicAuthentication)
}

func TestReadConfigFromYAMLBackwardsCompatibility(t *testing.T) {
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
	testConfigurationFile(t, "configuration.yml", contents, ReadConfigFromYAML, expectedConfigurationBasicAuthentication)
}

func TestWriteConfigToYAML(t *testing.T) {
	expectedContents := fmt.Sprintf(`CurrentContext: %s
Contexts:
  %s:
    Host: %s
    Timeout: %s
    Debug: %v
    %s:
      Username: %s
      Password: %s`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testTimeoutField,
		testDebugField,
		basicAuthenticationKeyYAML,
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
