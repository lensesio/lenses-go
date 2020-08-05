package config

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
	"github.com/lensesio/lenses-go/pkg/api"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/pflag"
)

//DefaultConfigFilepath the default config file path
var DefaultConfigFilepath = filepath.Join(api.DefaultConfigurationHomeDir, "lenses-cli.yml")

//Manager the config manager
var Manager *ConfigurationManager

//ConfigurationManager manager for config
type ConfigurationManager struct {
	Config *api.Config
	// flags below.
	CurrentContext, host, timeout, token, user, pass, kerberosConf, kerberosRealm, kerberosKeytab, kerberosCCache string
	insecure, debug                                                                                               bool

	Filepath string
}

/*
1. config home file not found, flags set, command run based on flags if authentication flags passed, don't save. (PASS)

2. config home file not found, neither auth flags passed, auto run configure command: cannot retrieve credentials, please configure below;
   save the configuration on the home file and run the command based on the passed configuration. (PASS)

3. config home file found, run the command based on that. (PASS)

4. config home file found, some flags set, set the filled flags to the config, override, and run the command, don't save. (PASS)

5. config home file not found, the --config flag passed and found, run the command based on that, don't save. (PASS)

6. config home file found, the --config flag passed and found, override the home's and run based on the --config, don't save. (PASS)

7. config home file not found, neither auth flags passed but command was one of "context" or "contexts" then show empty screen. (PASS)
  7.1 if "configure" command then must show the create configuration survey. (PASS)
*/

//NewConfigurationManager creates a configuration
func NewConfigurationManager(set *pflag.FlagSet) *ConfigurationManager {
	m := &ConfigurationManager{
		Config: &api.Config{
			Contexts: make(map[string]*api.ClientConfig),
		},
	}

	set.StringVar(&m.CurrentContext, "context", "", "Load specific environment, embedded configuration based on the configuration's 'Contexts'")

	set.StringVar(&m.host, "host", "", "Lenses host")
	// basic auth.

	// if --kerberos-conf set and not other kerberos-* flag set,
	// then kerberos with password method is selected based on the --user and --pass flags,
	// otherwise basic auth.
	set.StringVar(&m.user, "user", "", "User")
	set.StringVar(&m.pass, "pass", "", "Password")
	set.StringVar(&m.kerberosConf, "kerberos-conf", "", "krb5.conf")
	// if --kerberos-realm not set but --kerberos-config does then auth using kerberos with the default realm, otherwise using that realm.
	set.StringVar(&m.kerberosRealm, "kerberos-realm", "", "Kerberos realm")
	// if --kerberos-keytab & --kerberos-conf set then auth using kerberos keytab file.
	set.StringVar(&m.kerberosKeytab, "kerberos-keytab", "", "KeyTab file")
	// if --kerberos-ccache & --kerberos-conf set then auth from kerberos ccache file.
	set.StringVar(&m.kerberosCCache, "kerberos-ccache", "", "Kerberos keytab file")

	set.StringVar(&m.timeout, "timeout", "", "Timeout for the connection establishment")
	set.BoolVar(&m.insecure, "insecure", false, "All insecure http requests")
	set.StringVar(&m.token, "token", "", "Lenses auth token")
	set.BoolVar(&m.debug, "debug", false, "Print some information that are necessary for debugging")

	set.StringVar(&m.Filepath, "config", "", "Load or save the host, user, pass and debug fields from or to a configuration file (yaml or json)")
	return m
}

//NewEmptyConfigManager creates an empty configuration
func NewEmptyConfigManager() *ConfigurationManager {
	return &ConfigurationManager{
		Config: &api.Config{
			Contexts: make(map[string]*api.ClientConfig),
		},
	}
}

const currentContextEnvKey = "LENSES_CLI_CONTEXT"

//Load loads the configuration
func (m *ConfigurationManager) Load() (bool, error) {
	c := m.Config

	var found bool

	if m.Filepath != "" {
		// must read from file, otherwise fail.
		if err := api.TryReadConfigFromFile(m.Filepath, c); err != nil {
			return false, err
		}
		found = true
	} else if found = api.TryReadConfigFromCurrentWorkingDir(c); found {
	} else if found = api.TryReadConfigFromExecutable(c); found {
	} else if found = api.TryReadConfigFromHome(c); found {
	}
	// check --context flag (prio) and the configuration's one, if it's there and set the current context upfront.
	currentContext := c.CurrentContext
	currentContextChanged := false
	if flag := m.CurrentContext; flag != "" && flag != currentContext {
		currentContext = flag
		currentContextChanged = true
	} else if currentContext == "" {
		currentContext = api.DefaultContextKey
	}

	c.SetCurrent(currentContext)

	// authentication flags passed, override or set the particular authentication method.
	authFromFlags, authLoadedFromFlags := makeAuthFromFlags(m.user, m.pass, m.kerberosConf, m.kerberosRealm, m.kerberosKeytab, m.kerberosCCache)
	if authLoadedFromFlags {
		c.GetCurrent().Authentication = authFromFlags
	}

	// flags have always priority, so transfer any non-empty client configuration flag to the current,
	// so far we don't care about the configuration file found or not.
	c.GetCurrent().Fill(api.ClientConfig{
		Host:     m.host,
		Token:    m.token,
		Timeout:  m.timeout,
		Insecure: m.insecure,
		Debug:    m.debug,
	})

	if found {

		if currentContextChanged {
			// save the config, the current context changed.
			for _, v := range c.Contexts {
				DecryptPassword(v)
			}
			if err := m.Save(); err != nil {
				return false, err
			}
		} else {
			// check if loaded from flags, if so and we proceed then the password field goes empty.
			if !authLoadedFromFlags {
				// try to set the current context from *.env file or from system 's env variables,
				// if not empty, the env value has a priority over the configurated `CurrentContext`
				// but --context flag has a priority over all (look above).
				//
				// Note that the env variable will NOT change the `CurrentContext` field from the configuration file, by purpose.
				godotenv.Load()
				if envContext := strings.TrimSpace(os.Getenv(currentContextEnvKey)); envContext != "" {
					c.CurrentContext = envContext
				}
				for _, v := range c.Contexts {
					DecryptPassword(v)
				}
			}
		}
	}

	if c.CurrentContext != "" && !c.CurrentContextExists() {
		return false, fmt.Errorf("unknown context [%s] given, please use the `configure --context="+c.CurrentContext+" --reset`", c.CurrentContext)
	}

	return c.IsValid(), nil
}

//Save saves the configuration
func (m *ConfigurationManager) Save() error {
	c := m.Config.Clone() // copy the configuration so all changes here will not be present after the save().

	// we encrypt every password (main and contexts) because
	// they are decrypted on load, even if user didn't select to update a specific context.
	for _, v := range c.Contexts {
		v.FormatHost()
		if err := EncryptPassword(v); err != nil {
			return err
		}
	}

	// m.removeTokens()
	out, err := api.ConfigMarshalYAML(c)
	if err != nil { // should never happen.
		return fmt.Errorf("unable to marshal the configuration, error: [%v]", err)
	}

	if m.Filepath == "" {
		m.Filepath = DefaultConfigFilepath
	}
	directoryMode := os.FileMode(0750)
	// create any necessary directories.
	os.MkdirAll(filepath.Dir(m.Filepath), directoryMode)

	fileMode := os.FileMode(0600)
	// if file exists it overrides it.
	if err = ioutil.WriteFile(m.Filepath, out, fileMode); err != nil {
		return fmt.Errorf("unable to create the configuration file for your system, error: [%v]", err)
	}

	return nil
}

//EncryptPassword encrypts the password by provided client configuration
func EncryptPassword(cfg *api.ClientConfig) error {
	// if cfg.Kerberos.IsValid() && cfg.Password == "" { // if kerberos conf is valid and pass is empty here, skip encrypt, at least for now.
	// 	return nil
	// }
	if auth, ok := cfg.IsBasicAuth(); ok && auth.Password != "" {
		p, err := utils.EncryptString(auth.Password, cfg.Host)
		if err != nil {
			return err
		}

		auth.Password = p
		cfg.Authentication = auth
	} else if auth, ok := cfg.IsKerberosAuth(); ok {
		if withPass, ok := auth.WithPassword(); ok {
			p, err := utils.EncryptString(withPass.Password, cfg.Host)
			if err != nil {
				return err
			}

			withPass.Password = p
			auth.Method = withPass
			cfg.Authentication = auth
		}
	}

	return nil
}

//DecryptPassword decrypts the password by provided client configuration
func DecryptPassword(cfg *api.ClientConfig) {
	if auth, ok := cfg.IsBasicAuth(); ok && auth.Password != "" {
		p, _ := utils.DecryptString(auth.Password, cfg.Host)
		auth.Password = p
		cfg.Authentication = auth
	} else if auth, ok := cfg.IsKerberosAuth(); ok {
		if withPass, ok := auth.WithPassword(); ok {
			p, _ := utils.DecryptString(withPass.Password, cfg.Host)
			withPass.Password = p
			auth.Method = withPass
			cfg.Authentication = auth
		}
	}

}

//SetupConfigManager config manager
func SetupConfigManager(set *pflag.FlagSet) {
	Manager = NewConfigurationManager(set)
}

//Client used for the rest of the commands
var Client *api.Client

//SetupClient setups a new API client
func SetupClient() (err error) {
	Client, err = api.OpenConnection(*Manager.Config.GetCurrent())
	return
}

func makeAuthFromFlags(user, pass, kerberosConf, kerberosRealm, kerberosKeytab, kerberosCCache string) (api.Authentication, bool) {
	if kerberosConf != "" {
		auth := api.KerberosAuthentication{
			ConfFile: kerberosConf,
		}

		if kerberosKeytab == "" && kerberosCCache == "" && user != "" && pass != "" {
			auth.Method = api.KerberosWithPassword{Username: user, Password: pass, Realm: kerberosRealm}
		} else if kerberosKeytab != "" {
			auth.Method = api.KerberosWithKeytab{KeytabFile: kerberosKeytab}
		} else if kerberosCCache != "" {
			auth.Method = api.KerberosFromCCache{CCacheFile: kerberosCCache}
		} else {
			return nil, false
		}

		return auth, true
	}

	if user != "" && pass != "" {
		return api.BasicAuthentication{Username: user, Password: pass}, true
	}

	return nil, false
}
