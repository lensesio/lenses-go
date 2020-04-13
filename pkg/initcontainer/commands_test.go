package initcontainer

import (
	"bufio"
	"io/ioutil"
	"os"
	"strings"
	"testing"
	"github.com/stretchr/testify/assert"
)


func TestEnvWriterEnv(t *testing.T) {
	for _, v := range os.Environ() {
		if (strings.HasPrefix(v, "LENSES_") || strings.HasPrefix(v, "SECRET_")) {
			os.Unsetenv(strings.SplitN(v, "=", 2)[0])
		}
	}

	os.Setenv("LENSES_NONE_SENSITIVE_CONFIG", "ENV:you can see me")
	os.Setenv("SECRET_SECRET_STRING_CONFIG", "ENV:do not tell anyone")
	os.Setenv("SECRET_SECRET_BASE64_CONFIG", "ENV-base64:dGhpc2lzYXRlc3R0Cg==")
	os.Setenv("SECRET_SECRET_MOUNTED_BASE64_CONFIG", "ENV-mounted-base64:dGhpc2lzYXRlc3R0Cg==")

	var expected = []string{
		"export NONE_SENSITIVE_CONFIG=you can see me",
		"export SECRET_STRING_CONFIG=do not tell anyone",
		"export SECRET_BASE64_CONFIG=thisisatestt",
	}

	dir, _ := ioutil.TempDir("", "")
	file, _ := ioutil.TempFile(dir, "")

	vars, _ := AppConfigLoader()
	fileName := strings.TrimSuffix(strings.Trim(strings.ReplaceAll(file.Name(), dir, ""), "\\"), "/")
	write(vars, fileName, dir, "env")

	f, _ := os.Open(file.Name())
    scanner := bufio.NewScanner(f)
	
	scanner.Text()

	var actual []string

    for scanner.Scan() {
		actual = append(actual, scanner.Text())
	}
	
	assert.ElementsMatch(t, expected, actual)

	f2, _ := os.Open(dir + "/SECRET_MOUNTED_BASE64_CONFIG")
    scanner2 := bufio.NewScanner(f2)
	var fileContent string

	for scanner2.Scan() {
		fileContent = scanner2.Text()
	}

	assert.Equal(t, fileContent, "thisisatestt")

	defer os.RemoveAll(dir)
}

func TestEnvWriterProps(t *testing.T) {
	for _, v := range os.Environ() {
		if (strings.HasPrefix(v, "LENSES_") || strings.HasPrefix(v, "SECRET_")) {
			os.Unsetenv(strings.SplitN(v, "=", 2)[0])
		}
	}

	os.Setenv("LENSES_NONE_SENSITIVE_CONFIG", "ENV:you can see me")
	os.Setenv("SECRET_SECRET_STRING_CONFIG", "ENV:do not tell anyone")
	os.Setenv("SECRET_SECRET_BASE64_CONFIG", "ENV-base64:dGhpc2lzYXRlc3R0Cg==")
	os.Setenv("SECRET_SECRET_MOUNTED_BASE64_CONFIG", "ENV-mounted-base64:dGhpc2lzYXRlc3R0Cg==")

	var expected = []string{
		"none.sensitive.config=you can see me",
		"secret.string.config=do not tell anyone",
		"secret.base64.config=thisisatestt",
	}

	dir, _ := ioutil.TempDir("", "")
	file, _ := ioutil.TempFile(dir, "")

	vars, _ := AppConfigLoader()
	fileName := strings.TrimSuffix(strings.Trim(strings.ReplaceAll(file.Name(), dir, ""), "\\"), "/")
	write(vars, fileName, dir, "props")

	f, _ := os.Open(file.Name())
    scanner := bufio.NewScanner(f)
	
	scanner.Text()

	var actual []string

    for scanner.Scan() {
		actual = append(actual, scanner.Text())
	}
	
	assert.ElementsMatch(t, expected, actual)

	f2, _ := os.Open(dir + "/SECRET_MOUNTED_BASE64_CONFIG")
    scanner2 := bufio.NewScanner(f2)
	var fileContent string

	for scanner2.Scan() {
		fileContent = scanner2.Text()
	}

	assert.Equal(t, fileContent, "thisisatestt")

	defer os.RemoveAll(dir)
}