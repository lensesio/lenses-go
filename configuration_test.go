// Black-box testing for the configuration readers.
package lenses_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"testing"

	"github.com/landoop/lenses-go"
)

const testDebug = false

var expectedConfiguration = lenses.Configuration{
	Host:     "https://landoop.com",
	User:     "testuser",
	Password: "testpassword",
	Timeout:  "11s",
	Debug:    true,
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

func testConfigurationFile(t *testing.T, filename, contents string, reader func(string) lenses.Configuration) {
	t.Parallel()
	f, teardown := makeTestFile(t, filename)
	defer teardown()

	f.WriteString(contents)

	if got := reader(f.Name()); !reflect.DeepEqual(got, expectedConfiguration) {
		// Output format:
		/*
			configuration_test.go:51: error reading configuration from file: 'C:\Users\kataras\AppData\Local\Temp\configuration.json373943803'
			expected:
			lenses.Configuration{Host:"https://landoop.com", User:"testuser", Password:"testpassword", Timeout:"11s", Debug:true}
			but got:
			lenses.Configuration{Host:"", User:"testuser", Password:"testpassword", Timeout:"11s", Debug:true}
		*/
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
            "user": "%s",
            "password": "%s",
            "timeout": "%s",
            "debug": %v
        }`,
		expectedConfiguration.Host,
		expectedConfiguration.User,
		expectedConfiguration.Password,
		expectedConfiguration.Timeout,
		expectedConfiguration.Debug)
	testConfigurationFile(t, "configuration.json", contents, lenses.ReadConfigurationFromJSON)
}

func TestReadConfigurationFromYAML(t *testing.T) {
	contents := fmt.Sprintf(`
            Host: %s
            User: %s
            Password: %s
            Timeout: %s
            Debug: %v
        `,
		expectedConfiguration.Host,
		expectedConfiguration.User,
		expectedConfiguration.Password,
		expectedConfiguration.Timeout,
		expectedConfiguration.Debug)
	testConfigurationFile(t, "configuration.yml", contents, lenses.ReadConfigurationFromYAML)
}

func TestReadConfigurationFromTOML(t *testing.T) {
	contents := fmt.Sprintf(`
        Host = "%s"
        User = "%s"
        Password = "%s"
        Timeout = "%s"
        Debug = %v
    `,
		expectedConfiguration.Host,
		expectedConfiguration.User,
		expectedConfiguration.Password,
		expectedConfiguration.Timeout,
		expectedConfiguration.Debug)
	testConfigurationFile(t, "configuration.tml", contents, lenses.ReadConfigurationFromTOML)
}
