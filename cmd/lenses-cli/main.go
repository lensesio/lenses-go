// Package main provides the command line based tool for the Landoop's Lenses client REST API.
package main

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

var (
	// buildRevision is the build revision (docker commit string) but it's
	// available only on the build state, on the cli executable - via the "version" command.
	buildRevision = ""
	// buildTime is the build unix time (in nanoseconds), like the `buildRevision`,
	// this is available on after the build state, inside the cli executable - via the "version" command.
	//
	// Note that this BuildTime is not int64, it's type of string.
	buildTime = ""
)

var (
	configFilepath string
	config         lenses.Configuration
	client         *lenses.Client
)

const examplePrefix = `lenses-cli %s`

func exampleString(str string) string {
	return fmt.Sprintf(examplePrefix, str)
}

func tryLoadConfigurationFromFile(filename string) error {
	if filename == "" {
		return nil
	}

	cfg, err := lenses.TryReadConfigurationFromFile(filename)
	if err != nil {
		// here we could check for flags but we don't, if --config given then read from it,
		// otherwise fail in order to notify the user about that behavior.
		return err
	}
	fillConfig(cfg)
	return nil
}

func fillConfig(cfg lenses.Configuration) {
	p, _ := decryptString(cfg.Password, cfg.Host)
	cfg.Password = p
	config.Fill(cfg)
}

func tryLoadConfigurationFromCommonDirectories() {
	// search from the current working directory,
	// if not found then the executable's path
	// and if not found then try lookup from the home dir.
	// working directory and executable paths have priority over the home directory,
	// in order to make folder-based projects work as expected.
	if cfg, ok := lenses.TryReadConfigurationFromCurrentWorkingDir(); ok {
		fillConfig(cfg)
	} else if cfg, ok := lenses.TryReadConfigurationFromExecutable(); ok {
		fillConfig(cfg)
	} else if cfg, ok = lenses.TryReadConfigurationFromHome(); ok {
		fillConfig(cfg)
	}
}

var rootCmd = &cobra.Command{
	Use:                        "lenses-cli [command] [flags]",
	Example:                    exampleString(`sql --offsets --stats=2s "SELECT * FROM reddit_posts LIMIT 50"`),
	Short:                      "Lenses-cli is the command line client for the Landoop's Lenses REST API.",
	Version:                    lenses.Version,
	SilenceUsage:               true,
	Long:                       "lenses-cli - manage Lenses resources and developer workflow",
	SilenceErrors:              true,
	TraverseChildren:           true,
	SuggestionsMinimumDistance: 1,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) (err error) {
		if cmd.Name() == "docs" {
			cmd.ResetFlags()
			return nil // skip all that when in docs command.
		}

		if client != nil { // * client can be not empty in the future if we decide that we want sessions.
			return nil
		}

		if err = tryLoadConfigurationFromFile(configFilepath); err != nil {
			if cmd.Name() != "configure" {
				return
			}
			err = nil // skip the error if configure command, we need the `--config` to save, not to load then.
		}

		if !config.IsValid() {
			tryLoadConfigurationFromCommonDirectories()
		}
		//

		// if command is "configure" and the configuration is invalid at this point, don't give a failure,
		// let the configure command give a tutorial for user in order to create a configuration file.
		// Note that if clientConfig is valid and we are inside the configure command
		// then the configure will normally continue and save the valid configuration (that normally came from flags).
		if cmd.Name() == "configure" { // && !clientConfig.IsValid() {
			return nil
		}

		// if login, remove the token so setupClient will generate a new one and save it to the home dir/lenses-cli.yml.
		if cmd.Name() == "login" {
			config.Token = ""

			//  and fire any errors if host or user or pass are not there.
			if config.User == "" || config.Password == "" || config.Host == "" {
				// return fmt.Errorf("cannot retrieve credentials, please setup the configuration using the '%s' command first", "configure")
				//
				if err := newConfigureCommand().Execute(); err != nil {
					return err
				}

				// add a new line, so the login's session welcome messages has its place.
				fmt.Fprintln(cmd.OutOrStdout())
			}

			return nil
		}

		// --config missing, working dir, exec path and home dir doesn't contain any valid configuration
		// and not in the "configure" mode then give an error about missing flags:
		// The host and (token or (user, pass)) are the required flags.
		// Note that the `lenses.OpenConnection` will give errors if credentials missing
		// but let's catch them as soon as possible.
		if !config.IsValid() {
			return fmt.Errorf("cannot retrieve credentials, please setup using the '%s' command first", "configure")
		}

		// if config.Debug {
		// 	cmd.DebugFlags()
		// }

		// don't connect to the HTTP REST API when command is "live" (websocket).
		if cmd.Name() == "live" {
			return
		}

		return setupClient()
	},
}

func setupClient() (err error) {
	client, err = lenses.OpenConnection(config)
	if err == nil {
		config.Token = client.GetAccessToken()
	}

	return
}

// timeLayout defines the datetime layout for the `buildTime`.
const timeLayout = time.UnixDate

func buildVersionTmpl() string {
	/*
		- lenses-cli --version
		- version is the semantic version of the client package itself.
		- "build revision" is the build revision, available on build state, on the cli executable itself.
		- "build datetime" is originally the build time in unix nano seconds, formatted to human-readable text.
		- Output format:
			lenses-cli version 2.0
			>>>> build
						revision 27c7532fc6bf9c02bc7cf4575036ba0011f4c09a
						datetime Tu April 03 07:09:42 UTC 2018
						go       1.10
	*/
	buildTitle := ">>>> build" // if we ever want an emoji, there is one: \U0001f4bb
	tab := strings.Repeat(" ", len(buildTitle))

	// unix nanoseconds, as int64, to a human readable time, defaults to time.UnixDate, i.e:
	// Thu Mar 22 02:40:53 UTC 2018
	// but can be changed to something like "Mon, 01 Jan 2006 15:04:05 GMT" if needed.
	n, _ := strconv.ParseInt(buildTime, 10, 64)
	buildTimeStr := time.Unix(n, 0).Format(timeLayout)

	return `{{with .Name}}{{printf "%s " .}}{{end}}{{printf "version %s" .Version}}` +
		fmt.Sprintf("\n%s\n", buildTitle) +
		fmt.Sprintf("%s revision %s\n", tab, buildRevision) +
		fmt.Sprintf("%s datetime %s\n", tab, buildTimeStr) +
		fmt.Sprintf("%s go       %s\n", tab, runtime.Version())
}

var (
	errResourceNotFoundMessage string
	// more may come.
)

type errorMap map[error]string

func mapError(err error, messages errorMap) error {
	if messages == nil {
		return err
	}

	if errMsg, ok := messages[err]; ok {
		return fmt.Errorf(errMsg)
	}

	return err // otherwise just print the error as it's.
}

func main() {
	rootCmd.SetVersionTemplate(buildVersionTmpl())

	rootCmd.PersistentFlags().StringVar(&config.Host, "host", "", "--host=https://example.com")
	rootCmd.PersistentFlags().StringVar(&config.User, "user", "", "--user=MyUser")
	rootCmd.PersistentFlags().StringVar(&config.Timeout, "timeout", "", "--timeout=30s timeout for connection establishment")
	rootCmd.PersistentFlags().StringVar(&config.Password, "pass", "", "--pass=MyPassword")
	rootCmd.PersistentFlags().StringVar(&config.Token, "token", "", "--token=DSAUH321S%423#32$321ZXN")
	rootCmd.PersistentFlags().BoolVar(&config.Debug, "debug", false, "--debug=true will print some debug information that are necessary for debugging")

	rootCmd.PersistentFlags().StringVar(&configFilepath, "config", "", "--config loads/save the host, user, pass and debug options from a configuration file (yaml, toml or json)")

	if err := rootCmd.Execute(); err != nil {
		// catch any errors that should be described by the command that gave that error.
		// each errResourceXXXMessage should be declared inside the command,
		// they are global variables and that's because we don't want to get dirdy on each resource command, don't change it unless discussion.
		err = mapError(err, errorMap{
			lenses.ErrResourceNotFound: errResourceNotFoundMessage,
		})

		// always new line because of the unix terminal.
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
}
