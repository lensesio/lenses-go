package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
)

type configurationManager struct {
	config *lenses.Configuration
	flags  lenses.ClientConfiguration
	// TODO:
	user, pass, kerberosConf, kerberosRealm, kerberosKeytab, kerberosCCache string

	filepath string
	fromFile bool // if the configuration was SUCCESSFULLY loaded from --config flag.
}

func makeAuthFromFlags(user, pass, kerberosConf, kerberosRealm, kerberosKeytab, kerberosCCache string) (lenses.Authentication, bool) {
	if kerberosConf != "" {
		auth := lenses.KerberosAuthentication{
			ConfFile: kerberosConf,
		}

		if kerberosKeytab == "" && kerberosCCache == "" && user != "" && pass != "" {
			auth.Method = lenses.KerberosWithPassword{Username: user, Password: pass, Realm: kerberosRealm}
		} else if kerberosKeytab != "" {
			auth.Method = lenses.KerberosWithKeytab{KeytabFile: kerberosKeytab}
		} else if kerberosCCache != "" {
			auth.Method = lenses.KerberosFromCCache{CCacheFile: kerberosCCache}
		} else {
			return nil, false
		}

		return auth, true
	}

	if user == "" || pass != "" {
		return lenses.BasicAuthentication{Username: user, Password: pass}, true
	}

	return nil, false
}

func newConfigurationManager(cmd *cobra.Command) *configurationManager {
	m := &configurationManager{
		config: &lenses.Configuration{
			Contexts: make(map[string]*lenses.ClientConfiguration),
		},
	}

	set := cmd.PersistentFlags()

	set.StringVar(&m.config.CurrentContext, "context", "", "--context=dev load specific environment, embedded configuration based on the configuration's 'Contexts'")

	set.StringVar(&m.flags.Host, "host", "", "--host=https://example.com")
	// basic auth.

	// if --kerberos-conf set and not other kerberos-* flag set,
	// then kerberos with password method is selected based on the --user and --pass flags,
	// otherwise basic auth.
	set.StringVar(&m.user, "user", "", "--user=MyUser")
	set.StringVar(&m.pass, "pass", "", "--pass=MyPassword")
	set.StringVar(&m.kerberosConf, "kerberos-conf", "", "--kerberos-conf=krb5.conf")
	// if --kerberos-realm not set but --kerberos-config does then auth using kerberos with the default realm, otherwise using that realm.
	set.StringVar(&m.kerberosRealm, "kerberos-realm", "", "--kerberos-realm=kerberos.realm")
	// if --kerberos-keytab & --kerberos-conf set then auth using kerberos keytab file.
	set.StringVar(&m.kerberosKeytab, "kerberos-keytab", "", "--kerberos-keytab=/tmpl/krb5-my-keytab.txt")
	// if --kerberos-ccache & --kerberos-conf set then auth from kerberos ccache file.
	set.StringVar(&m.kerberosCCache, "kerberos-ccache", "", "--kerberos-ccache=/tmpl/krb5-ccache.txt")

	set.StringVar(&m.flags.Timeout, "timeout", "", "--timeout=30s timeout for the connection establishment")
	set.BoolVar(&m.flags.Insecure, "insecure", false, "--insecure=true")
	set.StringVar(&m.flags.Token, "token", "", "--token=DSAUH321S%423#32$321ZXN")
	set.BoolVar(&m.flags.Debug, "debug", false, "print some information that are necessary for debugging")

	set.StringVar(&m.filepath, "config", "", "load or save the host, user, pass and debug fields from or to a configuration file (yaml or json)")
	return m
}

const currentContextEnvKey = "LENSES_CLI_CONTEXT"

func (m *configurationManager) load() (bool, error) {
	c := m.config
	var found bool

	contextFlag := c.CurrentContext
	if m.filepath != "" {
		// must read from file, otherwise fail.
		if err := lenses.TryReadConfigurationFromFile(m.filepath, c); err != nil {
			return false, err
		}
		found = true
		m.fromFile = true
	} else if found = lenses.TryReadConfigurationFromCurrentWorkingDir(c); found {
	} else if found = lenses.TryReadConfigurationFromExecutable(c); found {
	} else if found = lenses.TryReadConfigurationFromHome(c); found {
	}

	if found {
		if contextFlag != "" && contextFlag != c.CurrentContext {
			// save the config, the current context changed.
			c.CurrentContext = contextFlag
			for _, v := range c.Contexts {
				decryptPassword(v)
			}
			if err := m.save(); err != nil {
				return false, err
			}
		} else {
			for _, v := range c.Contexts {
				decryptPassword(v)
			}

			// try to set the current context from *.env file or from system's env variables,
			// if not empty, the env value has a priority over the configurated `CurrentContext`
			// but --context flag has a priority over all (look above).
			//
			// Note that the env variable will NOT change the `CurrentContext` field from the configuration file, by purpose.
			godotenv.Load()
			if envContext := strings.TrimSpace(os.Getenv(currentContextEnvKey)); envContext != "" {
				c.CurrentContext = envContext
			}
		}
	}

	m.config.FillCurrent(m.flags)

	if m.config.CurrentContext != "" && !m.config.CurrentContextExists() {
		return false, fmt.Errorf("unknown context '%s' given, please use the `configure --context="+c.CurrentContext+" --reset`", c.CurrentContext)
	}

	return m.config.IsValid(), nil
}

func (m *configurationManager) save() error {
	c := m.config.Clone() // copy the configuration so all changes here will not be present after the save().

	// we encrypt every password (main and contexts) because
	// they are decrypted on load, even if user didn't select to update a specific context.
	for _, v := range c.Contexts {
		v.FormatHost()
		if err := encryptPassword(v); err != nil {
			return err
		}
	}

	// m.removeTokens()
	out, err := lenses.ConfigurationMarshalYAML(c)
	if err != nil { // should never happen.
		return fmt.Errorf("unable to marshal the configuration, error: %v", err)
	}

	if m.filepath == "" {
		m.filepath = defaultConfigFilepath
	}

	directoryMode := os.FileMode(0750)
	// create any necessary directories.
	os.MkdirAll(filepath.Dir(m.filepath), directoryMode)

	fileMode := os.FileMode(0600)
	// if file exists it overrides it.
	if err = ioutil.WriteFile(m.filepath, out, fileMode); err != nil {
		return fmt.Errorf("unable to create the configuration file for your system, error: %v", err)
	}

	return nil
}

// func (m *configurationManager) applyCompatibility() error {
// 	var (
// 		found     bool
// 		oldFormat lenses.ClientConfiguration // <>
// 	)

// 	// here we just fetch whatever is valid.
// 	if found = m.filepath != "" && (lenses.TryReadConfigurationFromFile(m.filepath, &oldFormat) == nil); found {
// 		m.fromFile = true
// 	} else if found = lenses.TryReadConfigurationFromCurrentWorkingDir(&oldFormat); found {
// 	} else if found = lenses.TryReadConfigurationFromExecutable(&oldFormat); found {
// 	} else if found = lenses.TryReadConfigurationFromHome(&oldFormat); found {
// 	}

// 	if !found || !oldFormat.IsValid() {
// 		return nil
// 	}

// 	decryptPassword(&oldFormat) // decrypt before save.
// 	if m.config.GetCurrent().Fill(oldFormat) {
// 		return m.save()
// 	}

// 	return nil
// }
