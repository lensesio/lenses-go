package lenses

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v2"
)

type (
	// Configuration contains the necessary information
	// that client needs to connect and talk to the lenses backend server.
	//
	// Configuration can be loaded via JSON or YAML.
	Configuration struct {
		// Host is the network address that your lenses backend is listening for incoming requests.
		Host string `json:"host" yaml:"Host" survey:"host"`

		// Authentication, in order to gain access using different kind of options.
		//
		// See `BasicAuthentication` and `KerberosAuthentication` or the example for more.
		Authentication Authentication `json:"-" yaml:"-" survey:"-"`

		// Token is the "X-Kafka-Lenses-Token" request header's value.
		// If not empty, overrides any `Authentication` settings.
		//
		// If `Token` is expired then all the calls will result on 403 forbidden error HTTP code
		// and a manual renewal will be demanded.
		//
		// For general-purpose usecase the recommendation is to let this field empty and
		// fill the `Authentication` field instead.
		Token string `json:"token,omitempty" yaml:"Token" survey:"-"`

		// Timeout specifies the timeout for connection establishment.
		//
		// Empty timeout value means no timeout.
		//
		// Such as "300ms", "-1.5h" or "2h45m".
		// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
		// Example: "5s" for 5 seconds, "5m" for 5 minutes and so on.
		Timeout string `json:"timeout,omitempty" yaml:"Timeout" survey:"timeout"`
		// Debug activates the debug mode, it logs every request, the configuration (except the `Password`)
		// and its raw response before decoded but after gzip reading.
		//
		// If this is enabled then printer's contents are not predicted to the end-user.
		// The output source is always the `os.Stdout` which 99.9% of the times means the terminal,
		// so use it only for debugging.
		//
		//
		// Defaults to false.
		Debug bool `json:"debug,omitempty" yaml:"Debug" survey:"debug"`
	} /* Why a whole Configuration struct while we could just pass those 3 params?
	Because we may need more fields in the future,
	and it's always a good practise to start like this on those type of packages.
	Another reason to not move those fields inside the Client itself is because
	we can load them via files, i.e in `OpenConnection`, we pass out options that are only runtime
	functions, they can't load via files.
	*/

)

var commaSep = []byte(",")

// ConfigurationJSONMarshal retruns the json string as bytes of the given `Configuration` structure.
func ConfigurationJSONMarshal(c Configuration) ([]byte, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	if c.Authentication == nil {
		return nil, nil
	}

	switch auth := c.Authentication.(type) {
	case BasicAuthentication:
		bb, err := json.Marshal(auth)
		if err != nil {
			return nil, err
		}
		bb = append([]byte(`,"basic_authentication":`), bb...)
		b = bytes.Replace(b, commaSep, append(bb, commaSep...), 1)
	case KerberosAuthentication:
	}

	return b, nil
}

func jsonUnmarshalConfiguration(b []byte, c *Configuration) error {
	// first unmarshal the known types.
	if err := json.Unmarshal(b, c); err != nil {
		return err
	}
	// second, get all.
	var raw map[string]json.RawMessage
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	// check if contains a valid authentication key.
	for k, v := range raw {
		isBasicAuth := k == "basic_authentication"
		isKerberosAuth := k == "kerberos_authentication"
		if isBasicAuth || isKerberosAuth {
			bb, err := v.MarshalJSON()
			if err != nil {
				return err
			}

			if isBasicAuth {
				var auth BasicAuthentication
				if err = json.Unmarshal(bb, &auth); err != nil {
					return err
				}
				c.Authentication = auth
				return nil
			}

			var auth KerberosAuthentication
			if err = json.Unmarshal(bb, &auth); err != nil {
				return err
			}
			c.Authentication = auth
			return nil
		}
	}

	// no new format found, let's do a backwards compatibility for "user" and "password" fields -> BasicAuthentication.
	if usernameJSON, passwordJSON := raw["user"], raw["password"]; len(usernameJSON) > 0 && len(passwordJSON) > 0 {
		// need to escape those "\"...\"".
		var auth BasicAuthentication
		if err := json.Unmarshal(usernameJSON, &auth.Username); err != nil {
			return err
		}

		if err := json.Unmarshal(passwordJSON, &auth.Password); err != nil {
			return err
		}

		c.Authentication = auth
		return nil
	}

	return fmt.Errorf("json: unknown or missing authentication key")
}

func yamlUnmarshalConfiguration(b []byte, c *Configuration) error {
	// first unmarshal the known types.
	if err := yaml.Unmarshal(b, c); err != nil {
		return err
	}
	// second, get all.
	var tree yaml.MapSlice
	err := yaml.Unmarshal(b, &tree)
	if err != nil {
		return err
	}

	// check if contains a valid authentication key.
	for _, v := range tree {
		if k, ok := v.Key.(string); ok {
			isBasicAuth := k == "BasicAuthentication"
			isKerberosAuth := k == "KerberosAuthentication"
			if isBasicAuth || isKerberosAuth {
				bb, err := yaml.Marshal(v.Value)
				if err != nil {
					return err
				}

				if isBasicAuth {
					var auth BasicAuthentication
					if err = yaml.Unmarshal(bb, &auth); err != nil {
						return err
					}
					c.Authentication = auth
					return nil
				}

				var auth KerberosAuthentication
				if err = yaml.Unmarshal(bb, &auth); err != nil {
					return err
				}
				c.Authentication = auth
				return nil
			}
		}
	}

	return fmt.Errorf("yaml: unknown or missing authentication key")
}

// FormatHost will try to make sure that the schema:host:port pattern is followed on the `Host` field.
func (c *Configuration) FormatHost() {
	if len(c.Host) == 0 {
		return
	}

	// remove last slash, so the API can append the path with ease.
	if c.Host[len(c.Host)-1] == '/' {
		c.Host = c.Host[0 : len(c.Host)-1]
	}

	portIdx := strings.LastIndexByte(c.Host, ':')

	schemaIdx := strings.Index(c.Host, "://")
	hasSchema := schemaIdx >= 0
	hasPort := portIdx > schemaIdx+1

	var port = "80"
	if hasPort {
		port = c.Host[portIdx+1:]
	}

	// find the schema based on the port.
	if !hasSchema {
		if port == "443" {
			c.Host = "https://" + c.Host
		} else {
			c.Host = "http://" + c.Host
		}
	} else if !hasPort {
		// has schema but not port.
		if strings.HasPrefix(c.Host, "https://") {
			port = "443"
		}
	}

	// finally, append the port part if it wasn't there.
	if !hasPort {
		c.Host += ":" + port
	}
}

// IsValid returns true if the configuration contains the necessary fields, otherwise false.
func (c *Configuration) IsValid() bool {
	if len(c.Host) == 0 {
		return false
	}

	c.FormatHost()

	return c.Host != "" && (c.Token != "" || c.Authentication != nil)
}

// Fill iterates over the "other" Configuration's fields
// it checks if a field is not empty,
// if it's then it sets the value to the "c" Configuration's particular field.
//
// It returns true if the final configuration is valid by calling the `IsValid`.
//
// Example of usage:
// Load configuration from flags directly, on the command run
// the file was loaded, if any, then try to check if flags given but give prioriy to flags(the "other").
func (c *Configuration) Fill(other Configuration) bool {
	if v := other.Host; v != "" && v != c.Host {
		c.Host = v
	}

	if other.Authentication != nil { // && c.Authentication == nil {
		c.Authentication = other.Authentication
	}

	if v := other.Token; v != "" && v != c.Token {
		c.Token = v
	}

	if v := other.Timeout; v != "" && v != c.Timeout {
		c.Timeout = v
	}

	if c.Debug != other.Debug {
		c.Debug = other.Debug
	}

	return c.IsValid()
}

// UnmarshalFunc is the most standard way to declare a Decoder/Unmarshaler to read the configurations and more.
// See `ReadConfiguration` and `ReadConfigurationFromFile` for more.
type UnmarshalFunc func(in []byte, outPtr *Configuration) error

// ReadConfiguration reads and decodes Configuration from an io.Reader based on a custom unmarshaler.
// This can be useful to read configuration via network or files (see `ReadConfigurationFromFile`).
// Sets the `outPtr`. Retruns a non-nil error on any unmarshaler's errors.
func ReadConfiguration(r io.Reader, unmarshaler UnmarshalFunc, outPtr *Configuration) error {
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	return unmarshaler(data, outPtr)
}

// ReadConfigurationFromFile reads and decodes Configuration from a file based on a custom unmarshaler,
// `ReadConfigurationFromJSON`, `ReadConfigurationFromYAML` and `ReadConfigurationFromTOML` are the internal users,
// but the end-developer can use any custom type of decoder to read a configuration file with ease using this function,
// but keep note that the default behavior of the fields depend on the existing unmarshalers, use these tag names to map
// your decoder's properties.
//
// Accepts the absolute or the relative path of the configuration file.
// Sets the `outPtr`. Retruns a non-nil error if parsing or decoding the file failed or file doesn't exist.
func ReadConfigurationFromFile(filename string, unmarshaler UnmarshalFunc, outPtr *Configuration) error {
	// get the abs
	// which will try to find the 'filename' from current working dir as well.
	absPath, err := filepath.Abs(filename)
	if err != nil {
		return err
	}

	f, err := os.Open(absPath)
	if err != nil {
		return err
	}

	err = ReadConfiguration(f, unmarshaler, outPtr)
	f.Close()
	return err
}

// TryReadConfigurationFromFile will try to read a specific file and unmarshal to `Configuration`.
// It will try to read it with one of these built'n lexers/formats:
// 1. JSON
// 2. YAML
// 3. TOML
func TryReadConfigurationFromFile(filename string, outPtr *Configuration) (err error) {
	tries := []UnmarshalFunc{
		jsonUnmarshalConfiguration,
		yamlUnmarshalConfiguration,
	}

	for _, unmarshaler := range tries {
		err = ReadConfigurationFromFile(filename, unmarshaler, outPtr)
		if err == nil { // if decoded without any issues, then return that as soon as possible.
			return
		}
	}

	return fmt.Errorf("configuration file '%s' does not exist or it is not formatted to a compatible document: JSON, YAML, TOML", filename)
}

var configurationPossibleFilenames = []string{
	"lenses-cli.yml", "lenses-cli.yaml", "lenses-cli.json",
	".lenses-cli.yml", ".lenses-cli.yaml", ".lenses-cli.json",
	"lenses.yml", "lenses.yaml", "lenses.json",
	".lenses.yml", ".lenses.yaml", ".lenses.json"} // no patterns in order to be easier to remove or modify these.

func lookupConfiguration(dir string, outPtr *Configuration) bool {
	for _, filename := range configurationPossibleFilenames {
		fullpath := filepath.Join(dir, filename)
		err := TryReadConfigurationFromFile(fullpath, outPtr)
		if err == nil {
			return true
		}
	}

	return false
}

// HomeDir returns the home directory for the current user on this specific host machine.
func HomeDir() (homeDir string) {
	u, err := user.Current() // ignore error handler.

	if u != nil && err == nil {
		homeDir = u.HomeDir
	}

	if homeDir == "" {
		homeDir = os.Getenv("HOME")
	}

	if homeDir == "" {
		if runtime.GOOS == "plan9" {
			homeDir = os.Getenv("home")
		} else if runtime.GOOS == "windows" {
			homeDir = os.Getenv("HOMEDRIVE") + os.Getenv("HOMEPATH")
			if homeDir == "" {
				homeDir = os.Getenv("USERPROFILE")
			}
		}
	}

	return
}

// DefaultConfigurationHomeDir is the default configuration system directory,
// by default it's the $HOME/.lenses directory.
var DefaultConfigurationHomeDir = filepath.Join(HomeDir(), ".lenses")

// TryReadConfigurationFromHome will try to read the `Configuration`
// from the current user's home directory/.lenses, the lookup is based on
// the common configuration filename pattern:
// lenses-cli.json, lenses-cli.yml, lenses-cli.yml or lenses.json, lenses.yml and lenses.tml.
func TryReadConfigurationFromHome(outPtr *Configuration) bool {
	return lookupConfiguration(DefaultConfigurationHomeDir, outPtr)
}

// TryReadConfigurationFromExecutable will try to read the `Configuration`
// from the (client's caller's) executable path that started the current process.
// The lookup is based on the common configuration filename pattern:
// lenses-cli.json, lenses-cli.yml, lenses-cli.yml or lenses.json, lenses.yml and lenses.tml.
func TryReadConfigurationFromExecutable(outPtr *Configuration) bool {
	executablePath, err := os.Executable()
	if err != nil {
		return false
	}

	executablePath = filepath.Dir(executablePath)

	return lookupConfiguration(executablePath, outPtr)
}

// TryReadConfigurationFromCurrentWorkingDir will try to read the `Configuration`
// from the current working directory, note that it may differs from the executable path.
// The lookup is based on the common configuration filename pattern:
// lenses-cli.json, lenses-cli.yml, lenses-cli.yml or lenses.json, lenses.yml and lenses.tml.
func TryReadConfigurationFromCurrentWorkingDir(outPtr *Configuration) bool {
	workingDir, err := os.Getwd()
	if err != nil {
		return false
	}

	return lookupConfiguration(workingDir, outPtr)
}

// ReadConfigurationFromJSON reads and decodes Configuration from a json file, i.e `configuration.json`.
//
// Accepts the absolute or the relative path of the configuration file.
// Parsing error will result to a panic.
// Error may occur when the file doesn't exists or is not formatted correctly.
func ReadConfigurationFromJSON(filename string, outPtr *Configuration) error {
	return ReadConfigurationFromFile(filename, jsonUnmarshalConfiguration, outPtr)
}

// ReadConfigurationFromYAML reads and decodes Configuration from a yaml file, i.e `configuration.yml`.
//
// Accepts the absolute or the relative path of the configuration file.
// Parsing error will result to a panic.
// Error may occur when the file doesn't exists or is not formatted correctly.
func ReadConfigurationFromYAML(filename string, outPtr *Configuration) error {
	return ReadConfigurationFromFile(filename, yamlUnmarshalConfiguration, outPtr)
}
