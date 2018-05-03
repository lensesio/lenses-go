package lenses

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/BurntSushi/toml"
	"gopkg.in/yaml.v2"
)

// Configuration contains the necessary information
// that client needs to connect and talk to the lenses backend server.
//
// Configuration can be loaded via JSON or YAML.
type Configuration struct {
	// Host is the network address that your lenses backend is listening for incoming requests.
	Host string `json:"host" yaml:"Host" toml:"Host" survey:"host"`

	// Auth fields
	// we need those in order to generate the access token.

	// User is your "user" field,
	User string `json:"user" yaml:"User" toml:"User" survey:"user"`
	// Password is your "password".
	Password string `json:"password,omitempty" yaml:"Password" toml:"Password" survey:"-"`

	// Token is the "X-Kafka-Lenses-Token" request header's value.
	// Overrides the `User` and `Password` settings.
	//
	// If `Token` is expired then all the calls will result on 403 forbidden error HTTP code
	// and a manual renewal will be demanded.
	//
	// For general-purpose usecase the recommendation is to let this field empty and
	// fill the `User` and `Password` instead.
	Token string `json:"token,omitempty" yaml:"Token" toml:"Token" survey:"-"`

	// Timeout specifies the timeout for connection establishment.
	//
	// Empty timeout value means no timeout.
	//
	// Such as "300ms", "-1.5h" or "2h45m".
	// Valid time units are "ns", "us" (or "µs"), "ms", "s", "m", "h".
	// Example: "5s" for 5 seconds, "5m" for 5 minutes and so on.
	Timeout string `json:"timeout,omitempty" yaml:"Timeout" toml:"Timeout" survey:"timeout"`
	// Debug activates the debug mode, it logs every request, the configuration (except the `Password`)
	// and its raw response before decoded but after gzip reading.
	//
	// If this is enabled then printer's contents are not predicted to the end-user.
	// The output source is always the `os.Stdout` which 99.9% of the times means the terminal,
	// so use it only for debugging.
	//
	//
	// Defaults to false.
	Debug bool `json:"debug,omitempty" yaml:"Debug" toml:"Debug" survey:"debug"` /*
	 Logging is hapenning trhough the `github.com/kataras/golog` and `pio` package,
	 which performs x3 times faster than the alternatives.
	 Zero performance cost if a logger is not responsible to actually print/write the message,
	 on `Debugf` not even the `fmt.Spritnf` is called in that case.
	 The user of the lenses client can change its output source(s) and even inject the log messaging
	 by using the `golog.Default` instance.
	*/
} /* Why a whole Configuration struct while we could just pass those 3 params?
Because we may need more fields in the future,
and it's always a good practise to start like this on those type of packages.
Another reason to not move those fields inside the Client itself is because
we can load them via files, i.e in `OpenConnection`, we pass out options that are only runtime
functions, they can't load via files.
*/

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

	return c.Host != "" && (c.Token != "" || (c.User != "" && c.Password != ""))
}

// Fill iterates over the current "c" Configuration's fields
// it checks if a field is empty or false,
// if it's then fill from the "other".
//
// It returns true if the final configuration is valid by calling the `IsValid`.
//
// Example of usage:
// Load configuration from flags directly, on the command run
// the flags are filled, if any, then try to load from files but give prioriy to flags(the current "c").
func (c *Configuration) Fill(other Configuration) bool {
	if c.Host == "" {
		c.Host = other.Host
	}

	if c.Token == "" {
		c.Token = other.Token
	}

	if c.User == "" {
		c.User = other.User
	}

	if c.Password == "" {
		c.Password = other.Password
	}

	if c.Timeout == "" {
		c.Timeout = other.Timeout
	}

	if !c.Debug {
		c.Debug = other.Debug
	}

	return c.IsValid()
}

// UnmarshalFunc is the most standard way to declare a Decoder/Unmarshaler to read the configurations and more.
// See `ReadConfiguration` and `ReadConfigurationFromFile` for more.
type UnmarshalFunc func(in []byte, outPtr interface{}) error

// ReadConfiguration reads and decodes Configuration from an io.Reader based on a custom unmarshaler.
// This can be useful to read configuration via network or files (see `ReadConfigurationFromFile`).
// Sets the `outPtr`. Retruns a non-nil error on any unmarshaler's errors.
func ReadConfiguration(r io.Reader, unmarshaler UnmarshalFunc, outPtr interface{}) error {
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
func ReadConfigurationFromFile(filename string, unmarshaler UnmarshalFunc, outPtr interface{}) error {
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
func TryReadConfigurationFromFile(filename string, outPtr interface{}) (err error) {
	tries := []UnmarshalFunc{
		json.Unmarshal,
		yaml.Unmarshal,
		toml.Unmarshal,
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
	"lenses-cli.yml", "lenses-cli.yaml", "lenses-cli.json", "lenses-cli.tml",
	".lenses-cli.yml", ".lenses-cli.yaml", ".lenses-cli.json", ".lenses-cli.tml",
	"lenses.yml", "lenses.yaml", "lenses.json", "lenses.tml",
	".lenses.yml", ".lenses.yaml", ".lenses.json", ".lenses.tml"} // no patterns in order to be easier to remove or modify these.

func lookupConfiguration(dir string, outPtr interface{}) bool {
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
func TryReadConfigurationFromHome(outPtr interface{}) bool {
	return lookupConfiguration(DefaultConfigurationHomeDir, outPtr)
}

// TryReadConfigurationFromExecutable will try to read the `Configuration`
// from the (client's caller's) executable path that started the current process.
// The lookup is based on the common configuration filename pattern:
// lenses-cli.json, lenses-cli.yml, lenses-cli.yml or lenses.json, lenses.yml and lenses.tml.
func TryReadConfigurationFromExecutable(outPtr interface{}) bool {
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
func TryReadConfigurationFromCurrentWorkingDir(outPtr interface{}) bool {
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
func ReadConfigurationFromJSON(filename string, outPtr interface{}) error {
	return ReadConfigurationFromFile(filename, json.Unmarshal, outPtr)
}

// ReadConfigurationFromYAML reads and decodes Configuration from a yaml file, i.e `configuration.yml`.
//
// Accepts the absolute or the relative path of the configuration file.
// Parsing error will result to a panic.
// Error may occur when the file doesn't exists or is not formatted correctly.
func ReadConfigurationFromYAML(filename string, outPtr interface{}) error {
	return ReadConfigurationFromFile(filename, yaml.Unmarshal, outPtr)
}

// ReadConfigurationFromTOML reads and decodess Configuration from a toml-compatible document file.
// Read more about toml's implementation at:
// https://github.com/toml-lang/toml
//
//
// Accepts the absolute or the relative path of the configuration file.
// Parsing error will result to a panic.
// Error may occur when the file doesn't exists or is not formatted correctly.
func ReadConfigurationFromTOML(filename string, outPtr interface{}) error {
	return ReadConfigurationFromFile(filename, toml.Unmarshal, outPtr)
}
