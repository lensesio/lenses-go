package lenses

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func TestBasicAuthenticationMarshalJSON(t *testing.T) {
	// order of struct fields matter here.
	expectedConfig := fmt.Sprintf(`{"currentContext":"%s","contexts":{"%s":{"host":"%s","%s":{"username":"%s","password":"%s"},"timeout":"%s","insecure":%v,"debug":%v}}}`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		basicAuthenticationKeyJSON,
		testUsernameField,
		testPasswordField,
		testTimeoutField,
		testInsecureField,
		testDebugField,
	)

	gotConfig, err := ConfigurationMarshalJSON(Config{
		CurrentContext: testCurrentContextField,
		Contexts: map[string]*ClientConfig{
			testCurrentContextField: &ClientConfig{
				Host:           testHostField,
				Authentication: testBasicAuthenticationField,
				Timeout:        testTimeoutField,
				Insecure:       testInsecureField,
				Debug:          testDebugField,
			},
		},
	})

	if err != nil {
		t.Fatal(err)
	}

	if expected, got := strings.TrimSpace(expectedConfig), strings.TrimSpace(string(gotConfig)); expected != got {
		t.Fatalf("expected raw json configuration to be:\n'%s'\nbut got:\n'%s'", expected, got)
	}
}

func testKerberosAuthenticationJSON(t *testing.T, expectedAuthStr string, expectedMethod KerberosAuthenticationMethod) {
	expectedConfigStr := strings.TrimSpace(fmt.Sprintf(`{"currentContext":"%s","contexts":{"%s":{"host":"%s","%s":{"%s":"%s",%s},"timeout":"%s","insecure":%v,"debug":%v}}}`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		kerberosAuthenticationKeyJSON,
		kerberosConfFileKeyJSON,
		testKerberosConfFileField,
		expectedAuthStr,
		testTimeoutField,
		testInsecureField,
		testDebugField,
	))

	expectedConfig := Config{
		CurrentContext: testCurrentContextField,
		Contexts: map[string]*ClientConfig{
			testCurrentContextField: &ClientConfig{
				Host:           testHostField,
				Authentication: KerberosAuthentication{ConfFile: testKerberosConfFileField, Method: expectedMethod},
				Timeout:        testTimeoutField,
				Insecure:       testInsecureField,
				Debug:          testDebugField,
			},
		},
	}

	gotConfig, err := ConfigurationMarshalJSON(expectedConfig)

	if err != nil {
		t.Fatal(err)
	}

	if expected, got := expectedConfigStr, strings.TrimSpace(string(gotConfig)); expected != got {
		t.Fatalf("expected raw json configuration to be:\n'%s'\nbut got:\n'%s'", expected, got)
	}

	var gotUnmarshaledConfig Config
	if err := ConfigurationUnmarshalJSON([]byte(expectedConfigStr), &gotUnmarshaledConfig); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(expectedConfig, gotUnmarshaledConfig) {
		onlyAuthExpected := expectedConfig.Contexts[testCurrentContextField].Authentication
		if _, hasContext := gotUnmarshaledConfig.Contexts[testCurrentContextField]; !hasContext {
			t.Fatalf("expected configuration has a context with key '%s' but gotten unmarshaled has none, more:\n%#+v\nvs\n%#+v", testCurrentContextField, expectedConfig, gotUnmarshaledConfig)
		}
		onlyAuthGot := gotUnmarshaledConfig.Contexts[testCurrentContextField].Authentication
		t.Fatalf("expected configuration structure after unmarshal the succeed marshaled:\n%#+v\nbut got:\n%#+v", onlyAuthExpected, onlyAuthGot)
	}
}
func TestKerberosAuthenticationJSON_WithPassword(t *testing.T) {
	expectedAuthStr := fmt.Sprintf(`"%s":{"username":"%s","password":"%s","realm":"%s"}`,
		kerberosWithPasswordMethodKeyJSON,
		testUsernameField,
		testPasswordField,
		testKerberosRealmField,
	)

	testKerberosAuthenticationJSON(t, expectedAuthStr, testKerberosMethodWithPasswordField)
}

func TestKerberosAuthenticationJSON_WithKeytab(t *testing.T) {
	expectedAuthStr := fmt.Sprintf(`"%s":{"username":"%s","realm":"%s","keytabFile":"%s"}`,
		kerberosWithKeytabMethodKeyJSON,
		testUsernameField,
		testKerberosRealmField,
		testKerberosKeytabField,
	)

	testKerberosAuthenticationJSON(t, expectedAuthStr, testKerberosMethodWithKeytabField)
}

func TestKerberosAuthenticationJSON_FromCCache(t *testing.T) {
	expectedAuthStr := fmt.Sprintf(`"%s":{"ccacheFile":"%s"}`,
		kerberosFromCCacheMethodKeyJSON,
		testKerberosCCacheField,
	)

	testKerberosAuthenticationJSON(t, expectedAuthStr, testKerberosMethodFromCCacheField)
}
