// Package main provides the command line based tool for the Landoop's Lenses client REST API.
package main

import (
	"fmt"
	"os"
	"time"

	"github.com/landoop/lenses-go"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	// buildRevision is the build revision (docker commit string or git rev-parse HEAD) but it's
	// available only on the build state, on the cli executable - via the "--version" flag.
	buildRevision = ""
	// buildTime is the build unix time (in seconds since 1970-01-01 00:00:00 UTC), like the `buildRevision`,
	// this is available on after the build state, inside the cli executable - via the "--version" flag.
	//
	// Note that this buildTime is not int64, it's type of string.
	buildTime = fmt.Sprintf("%d", time.Now().Unix())
)

var (
	app = &bite.Application{
		Name:            "lenses-cli",
		Description:     "Lenses-cli is the command line client for the Landoop's Lenses REST API.",
		Version:         lenses.Version,
		PersistentFlags: setupConfigManager,
		ShowSpinner:     false,
		Setup:           setup,
	}

	configManager *configurationManager
	client        *lenses.Client
)

func setupConfigManager(set *pflag.FlagSet) {
	configManager = newConfigurationManager(set)
}

func setupClient() (err error) {
	client, err = lenses.OpenConnection(*configManager.config.GetCurrent())
	return
}

func setup(cmd *cobra.Command, args []string) error {
	ok, err := configManager.load()
	// if command is "configure" and the configuration is invalid at this point, don't give a failure,
	// let the configure command give a tutorial for user in order to create a configuration file.
	// Note that if clientConfig is valid and we are inside the configure command
	// then the configure will normally continue and save the valid configuration (that normally came from flags).
	if name := cmd.Name(); name == "configure" || name == "context" || name == "contexts" {
		return nil
	}

	// it's not nil, if context does not exist then it would throw an error.
	currentConfig := configManager.config.GetCurrent()
	for !ok {
		if err != nil {
			return err
		}

		if currentConfig.Debug {
			fmt.Fprintf(cmd.OutOrStdout(), "%#+v\n", *currentConfig)
		}

		fmt.Fprintln(cmd.OutOrStderr(), "cannot retrieve credentials, please configure below")
		configureCmd := newConfigureCommand()
		// disable any flags passed on the parent command before execute.
		configureCmd.DisableFlagParsing = true
		if err = configureCmd.Execute(); err != nil {
			return err
		}

		ok, err = configManager.load()
	}

	// if login, remove the token so setupClient will generate a new one and save it to the home dir/lenses-cli.yml.
	if cmd.Name() == "login" {
		currentConfig.Token = ""

		if basicAuth, isBasicAuth := currentConfig.Authentication.(lenses.BasicAuthentication); isBasicAuth {
			//  and fire any errors if host or user or pass are not there.
			if currentConfig.Host == "" || basicAuth.Username == "" || basicAuth.Password == "" {
				// return fmt.Errorf("cannot retrieve credentials, please setup the configuration using the '%s' command first", "configure")
				//
				if err := newConfigureCommand().Execute(); err != nil {
					return err
				}

				// add a new line, so the login's session welcome messages has its place.
				fmt.Fprintln(cmd.OutOrStdout())
			}
		}

		return nil
	}

	// don't connect to the HTTP REST API when command is "live" (websocket).
	if cmd.Name() == "live" {
		return nil
	}

	return setupClient()
}

const (
	errResourceNotFoundMessage      = 404 // 404
	errResourceNotAccessibleMessage = 403 // 403
	errResourceNotGoodMessage       = 400 // 400
	errResourceInternal             = 500 // 500
)

func main() {
	if buildRevision != "" {
		app.HelpTemplate = bite.HelpTemplate{
			BuildRevision:        buildRevision,
			BuildTime:            buildTime,
			ShowGoRuntimeVersion: true,
		}
	}

	if err := app.Run(os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
