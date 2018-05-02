package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

// Configuration is the CLI's configuration.
// The `Contexts` map of string and lenses client configuration values can be filled to map different environments.
type Configuration struct {
	// lenses.Configuration `yaml:",inline"`
	CurrentContext string                           `yaml:"CurrentContext"`
	Contexts       map[string]*lenses.Configuration `yaml:"Contexts"`
}

type configurationManager struct {
	config   *Configuration
	flags    lenses.Configuration
	filepath string
	fromFile bool // if the configuration was SUCCESSFULLY loaded from --config flag.
}

func newConfigurationManager(cmd *cobra.Command) *configurationManager {
	m := &configurationManager{
		config: &Configuration{
			Contexts: make(map[string]*lenses.Configuration),
		},
	}

	cmd.PersistentFlags().StringVar(&m.config.CurrentContext, "context", "", "--context=dev load specific environment, embedded configuration based on the configuration's 'Contexts'")

	cmd.PersistentFlags().StringVar(&m.flags.Host, "host", "", "--host=https://example.com")
	cmd.PersistentFlags().StringVar(&m.flags.User, "user", "", "--user=MyUser")
	cmd.PersistentFlags().StringVar(&m.flags.Timeout, "timeout", "", "--timeout=30s timeout for the connection establishment")
	cmd.PersistentFlags().StringVar(&m.flags.Password, "pass", "", "--pass=MyPassword")
	cmd.PersistentFlags().StringVar(&m.flags.Token, "token", "", "--token=DSAUH321S%423#32$321ZXN")
	cmd.PersistentFlags().BoolVar(&m.flags.Debug, "debug", false, "print some information that are necessary for debugging")

	cmd.PersistentFlags().StringVar(&m.filepath, "config", "", "load or save the host, user, pass and debug fields from or to a configuration file (yaml, toml or json)")
	return m
}

// isValid returns the result of the contexts' lenses.Configuration#IsValid.
func (m *configurationManager) isValid() bool {
	// for a whole configuration to be valid we need to check each contexts' configs as well.
	for _, cfg := range m.config.Contexts {
		if !cfg.IsValid() {
			return false
		}
	}

	return len(m.config.Contexts) > 0
}

func (m *configurationManager) fillCurrent(cfg lenses.Configuration) {
	c := m.config

	context := c.CurrentContext

	if _, ok := c.Contexts[context]; !ok {
		if cfg.IsValid() {
			c.Contexts[context] = &cfg
		}
	} else {
		c.Contexts[context].Fill(cfg)
	}
}

func (m *configurationManager) currentContextExists() bool {
	_, ok := m.config.Contexts[m.config.CurrentContext]
	return ok
}

func (m *configurationManager) setCurrent(currentContext string) {
	m.config.CurrentContext = currentContext
}

func (m *configurationManager) getCurrent() *lenses.Configuration {
	if c, has := m.config.Contexts[m.config.CurrentContext]; has {
		return c
	}

	c := new(lenses.Configuration)
	if m.config.CurrentContext == "" {
		m.config.CurrentContext = "master" // the default one if missing.
	}
	m.config.Contexts[m.config.CurrentContext] = c
	return c
}

func (m *configurationManager) removeTokens() {
	for _, v := range m.config.Contexts {
		v.Token = ""
	}
}

// returns true if found and removed, otherwise false.
func (m *configurationManager) removeContext(contextName string) bool {
	if _, ok := m.config.Contexts[contextName]; ok {
		delete(m.config.Contexts, contextName)
		if err := m.save(); err != nil {
			return false
		}
		return true
	}

	return false
}

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
		}

	}

	m.fillCurrent(m.flags)

	if m.config.CurrentContext != "" && !m.currentContextExists() {
		return false, fmt.Errorf("unknown context '%s' given, please use the `configure --context="+c.CurrentContext+" --reset`", c.CurrentContext)
	}

	return m.isValid(), nil
}

func (m *configurationManager) clone() Configuration {
	c := Configuration{CurrentContext: m.config.CurrentContext}
	c.Contexts = make(map[string]*lenses.Configuration, len(m.config.Contexts))
	for k, v := range m.config.Contexts {
		vCopy := *v
		c.Contexts[k] = &vCopy
	}

	return c
}

func (m *configurationManager) save() error {
	c := m.clone() // copy the configuration so all changes here will not be present after the save().

	// we encrypt every password (main and contexts) because
	// they are decrypted on load, even if user didn't select to update a specific context.
	for _, v := range c.Contexts {
		if err := encryptPassword(v); err != nil {
			return err
		}
	}

	// m.removeTokens()
	out, err := yaml.Marshal(c)
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

func (m *configurationManager) applyCompatibility() error {
	var (
		found     bool
		oldFormat lenses.Configuration // <>
	)

	// here we just fetch whatever is valid.
	if found = m.filepath != "" && (lenses.TryReadConfigurationFromFile(m.filepath, &oldFormat) == nil); found {
		m.fromFile = true
	} else if found = lenses.TryReadConfigurationFromCurrentWorkingDir(&oldFormat); found {
	} else if found = lenses.TryReadConfigurationFromExecutable(&oldFormat); found {
	} else if found = lenses.TryReadConfigurationFromHome(&oldFormat); found {
	}

	if !found || !oldFormat.IsValid() { // do not proceed if it's not valid because it will fill the current context even if new config exists.
		return nil
	}

	decryptPassword(&oldFormat) // decrypt before save.
	if m.getCurrent().Fill(oldFormat) {
		return m.save()
	}

	return nil
}
