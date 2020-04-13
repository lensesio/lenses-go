package initcontainer

import (
	"os"
	"strings"
	"fmt"
	"encoding/base64"
	"github.com/spf13/cobra"
	golog "github.com/kataras/golog"
	"github.com/landoop/lenses-go/pkg/utils"
)

var secretPrefix = "SECRET_"
var lensesPrefix = "LENSES_"
var base64Str = "base64"
var mountedStr = "mounted"
var envPrefix = "ENV"

type (
	appVar struct {
		Prefix	string
		EnvKey	string
		Key	string
		Value	string
	}
)

// NewInitConCommand creates `init-container configure` command
func NewInitConCommand() *cobra.Command {

	var appFile, appDir, output string

	cmd := &cobra.Command{
		Use:   "init-container",
		Short: `Create files to source from environment variables`,
		Long: `
This command is intend for kubernetes init-containers. Sensitive data is inject as 
environment variables by kubernetes secrets and then written to a file. 

Two kind of secrets are supported: string secrets and file secrets.

Separate files will be created for each secret that is tagged as "mounted". The name of the file that
will be created is the value of the name of the environment variable (without the prefix).

Base64 and plaintext are supported for each secret's payload.

Secrets must be set as environment variables and prefixed with "SECRET_".
Non sensitive app configurations must be set as environment variables and prefixed with "LENSES_"

Prefixes are stripped.

Examples of environment variables for secrets:

	SECRET_TEST1=ENV:thisisatest
	SECRET_TEST2=ENV-mounted-base64:dGhpc2lzYXRlc3R0Cg==
	SECRET_TEST3=ENV-base64:dGhpc2lzYXRlc3R0Cg==

Examples for non sensitive environment varibales
	
	LENSES_MY_APP_CONFIG_KEY=ENV:my-nonsensitive-value

If the output is set to "env" the file contents can be sourced

	export TEST1=thisisatest

If the output is set to "props" the file contents are set as a properties. The environment
variable names are converted to lower case and any "_" are converted to "." .

	my.app.config.key=my-nonsensitive-value
`,
		Example: `
init-container app-config --dir=. --file=app-config --output=props"
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			vars, err := AppConfigLoader()
			if err != nil {
				return err
			}

			write(vars, appFile, appDir, output)

			return nil
		},
	}


	cmd.Flags().StringVar(&appFile, "file", "app-config", "The name of the file to write the app config and secrets to")
	cmd.Flags().StringVar(&appDir, "dir", ".", "The directory to write files to (for both string and base64 secrets")
	cmd.Flags().StringVar(&output, "output", "props", "Output type for contents of file. ENV for env files to be source, PROPS for props. Default is props")

	return cmd
}


func decodeBase64(data string, key string) string {
	golog.Infof("Decoding base64 variable [%s]", key)
	decodedValue, err := base64.StdEncoding.DecodeString(data)

	if err != nil {
		golog.Errorf("Error decoding base64 value for key [%s]: %s", key, err.Error())
	}
	return strings.TrimSuffix(string(decodedValue), "\n")
}

func write(vars []appVar, file, dir string, output string) error {
	var data []string

	for _, s := range vars {

		var content string
		// provider:value or reference
		// provider-base64:value or reference
		// provider-[mounted]:value or reference
		// provider-[mounted]-base64:value or reference
		var value = strings.SplitN(s.Value, ":", 2)
		valueMetadata := strings.SplitN(value[0], "-", 3)
		provider := strings.ToUpper(valueMetadata[0])

		if (len(valueMetadata) > 0 && provider != envPrefix) {
			golog.Warnf("Env value format metadata must being with [%s] for [%s], discarding", envPrefix, s.EnvKey)
			continue
		}

		if (len(value) > 1) {
			content = value[1]
		} else {
			content = value[0]
		}

		// env-mounted-base64
		if len(valueMetadata) == 3 && valueMetadata[2] == base64Str {
			content = decodeBase64(content, s.EnvKey)
		}

		// env-base64
		if len(valueMetadata) == 2 && valueMetadata[1] == base64Str {
			content = decodeBase64(content, s.EnvKey)
		}

		// env-mounted
		if len(valueMetadata) > 1 && valueMetadata[1] == mountedStr {
			err := utils.WriteStringFile(dir+"/"+s.Key, []string{content})
			if err != nil {
				golog.Errorf("Error writing file for key [%s]: %s", s.EnvKey, err.Error())
				continue
			}
			continue
		}

		if (strings.ToLower(output) == "env") {
			data = append(data, fmt.Sprintf("export %s=%s", s.Key, content))
		} else {
			data = append(data, fmt.Sprintf("%s=%s", strings.ToLower(strings.ReplaceAll(s.Key, "_", ".")), content))
		}
	}

	if len(data) > 0 {
		golog.Infof("Writing file [%s] to be sourced", file)
		err := utils.WriteStringFile(dir+"/"+file, data)
		if err != nil {
			return err
		}
	}

	return nil
}


func getVars(prefix string) []string {
	// get secret vars
	var results []string
	golog.Infof("Looking for environment variables prefixed with [%s]", prefix)
	vars := os.Environ()

	for _, v := range vars {
		if (strings.HasPrefix(v, prefix)) {
			golog.Infof("Found environment variable [%s]", strings.SplitN(v, "=", 2)[0])
			results = append(results, v)
		}
	}

	if (len(results) == 0) {
		golog.Warnf("No environment variables found for prefix [%s]", prefix)
	}

	return results
}

// AppConfigLoader retrieves secrets and app config key values from environment variables
func AppConfigLoader() ([]appVar, error) {

	var results []appVar

	// load secrets from environment or file
	sVars := getVars(secretPrefix)
	aVars := getVars(lensesPrefix)

	for _, v := range sVars {
		split := strings.SplitN(v, "=", 2)
		envKey := split[0]
		key := strings.Replace(envKey, secretPrefix, "", 1)
		value := split[1]
		results = append(results, appVar{Prefix: secretPrefix, EnvKey: envKey, Key: key, Value: value})
	}

	for _, v := range aVars {
		split := strings.SplitN(v, "=", 2)
		envKey := split[0]
		key := strings.Replace(envKey, lensesPrefix, "", 1)
		value := split[1]
		results = append(results, appVar{Prefix: lensesPrefix, EnvKey: envKey, Key: key, Value: value})
	}

	return results, nil
}

