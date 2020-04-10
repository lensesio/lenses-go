package test

import (
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
)

//SetupMasterContext add a new context named "master" in Config
func SetupMasterContext() {
	SetupContext(api.DefaultContextKey, ClientConfig, auth)
}

//SetupContext add a new context in Config
func SetupContext(contextName string, clientConfig api.ClientConfig, basicAuth api.BasicAuthentication) {
	SetupConfigManager()
	config.Manager.Config.AddContext(contextName, &clientConfig)
	config.Manager.Config.SetCurrent(contextName)
	config.Manager.Config.GetCurrent().Authentication = basicAuth
}

//SetupConfigManager setup a new empty config manager if not done already
func SetupConfigManager() {
	if config.Manager == nil {
		config.Manager = config.NewEmptyConfigManager()
	}
}

//ResetConfigManager reset the config manager
func ResetConfigManager() {
	config.Manager = nil
}
