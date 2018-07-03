package lenses

import (
	"strings"
	"testing"

	"fmt"
)

func TestConfigurationMarshalYAML_BasicAuthentication(t *testing.T) {
	expectedConfig := fmt.Sprintf(`
CurrentContext: %s
Contexts:
  %s:
    Host: %s
    Timeout: %s
    Insecure: %v
    Debug: %v
    BasicAuthentication:
      Username: %s
      Password: %s
		`,
		testCurrentContextField,
		testCurrentContextField,
		testHostField,
		testTimeoutField,
		testInsecureField,
		testDebugField,
		testUsernameField,
		testPasswordField,
	)

	gotConfig, err := ConfigurationMarshalYAML(Configuration{
		CurrentContext: testCurrentContextField,
		Contexts: map[string]*ClientConfiguration{
			testCurrentContextField: &ClientConfiguration{
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
		t.Fatalf("expected raw yaml configuration to be:\n'%s'\nbut got:\n'%s'", expected, got)
	}
}
