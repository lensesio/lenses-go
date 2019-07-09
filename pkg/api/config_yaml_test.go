package api

import (
	"fmt"
	"reflect"
	"testing"
)

func TestBasicAuthenticationMarshalYAML(t *testing.T) {
	expectedConfig := fmt.Sprintf(`CurrentContext: %s
Contexts:
  %s:
    Host: %s
    Timeout: %s
    Insecure: %v
    Debug: %v
    %s:
      Username: %s
      Password: %s`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testTimeoutField,
		testInsecureField,
		testDebugField,
		basicAuthenticationKeyYAML,
		testUsernameField,
		testPasswordField,
	)

	gotConfig, err := ConfigMarshalYAML(Config{
		CurrentContext: testCurrentContextField,
		Contexts: map[string]*ClientConfig{
			testCurrentContextField: {
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

	if expected, got := expectedConfig, string(gotConfig); expected != got {
		t.Fatalf("expected raw yaml configuration to be:\n'%s'\nbut got:\n'%s'", expected, got)
	}
}

// tests marshal and unmarshal.
func testKerberosAuthenticationYAML(t *testing.T, expectedAuthStr string, expectedMethod KerberosAuthenticationMethod) {
	expectedConfigStr := fmt.Sprintf(`CurrentContext: %s
Contexts:
  %s:
    Host: %s
    Timeout: %s
    Insecure: %v
    Debug: %v
    %s:
      %s: %s%s`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testTimeoutField,
		testInsecureField,
		testDebugField,
		kerberosAuthenticationKeyYAML,
		kerberosConfFileKeyYAML,
		testKerberosConfFileField,
		expectedAuthStr,
	)

	expectedConfig := Config{
		CurrentContext: testCurrentContextField,
		Contexts: map[string]*ClientConfig{
			testCurrentContextField: {
				Host:           testHostField,
				Authentication: KerberosAuthentication{ConfFile: testKerberosConfFileField, Method: expectedMethod},
				Timeout:        testTimeoutField,
				Insecure:       testInsecureField,
				Debug:          testDebugField,
			},
		},
	}

	gotConfig, err := ConfigMarshalYAML(expectedConfig)

	if err != nil {
		t.Fatal(err)
	}

	if expected, got := expectedConfigStr, string(gotConfig); expected != got {
		t.Fatalf("expected raw yaml configuration to be:\n'%s'\nbut got:\n'%s'", expected, got)
	}

	var gotUnmarshaledConfig Config
	if err := ConfigUnmarshalYAML([]byte(expectedConfigStr), &gotUnmarshaledConfig); err != nil {
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
func TestKerberosAuthenticationYAML_WithPassword(t *testing.T) {
	expectedAuthStr := fmt.Sprintf(`
      %s:
        Username: %s
        Password: %s
        Realm: %s`,
		kerberosWithPasswordMethodKeyYAML,
		testUsernameField,
		testPasswordField,
		testKerberosRealmField,
	)

	testKerberosAuthenticationYAML(t, expectedAuthStr, testKerberosMethodWithPasswordField)
}

func TestKerberosAuthenticationYAML_WithKeytab(t *testing.T) {
	expectedAuthStr := fmt.Sprintf(`
      %s:
        Username: %s
        Realm: %s
        KeytabFile: %s`,
		kerberosWithKeytabMethodKeyYAML,
		testUsernameField,
		testKerberosRealmField,
		testKerberosKeytabField,
	)

	testKerberosAuthenticationYAML(t, expectedAuthStr, testKerberosMethodWithKeytabField)
}

func TestKerberosAuthenticationYAML_FromCCache(t *testing.T) {
	expectedAuthStr := fmt.Sprintf(`
      %s:
        CCacheFile: %s`,
		kerberosFromCCacheMethodKeyYAML,
		testKerberosCCacheField,
	)

	testKerberosAuthenticationYAML(t, expectedAuthStr, testKerberosMethodFromCCacheField)
}
