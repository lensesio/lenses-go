// Package main provides the command line based tool for the Landoop's Lenses client REST API.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg/acl"
	"github.com/landoop/lenses-go/pkg/alert"
	"github.com/landoop/lenses-go/pkg/api"
	"github.com/landoop/lenses-go/pkg/audit"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/connector"
	"github.com/landoop/lenses-go/pkg/export"
	imports "github.com/landoop/lenses-go/pkg/import"
	"github.com/landoop/lenses-go/pkg/logs"
	"github.com/landoop/lenses-go/pkg/management"
	"github.com/landoop/lenses-go/pkg/policy"
	"github.com/landoop/lenses-go/pkg/processor"
	"github.com/landoop/lenses-go/pkg/quota"
	"github.com/landoop/lenses-go/pkg/schema"
	"github.com/landoop/lenses-go/pkg/secret"
	"github.com/landoop/lenses-go/pkg/shell"
	"github.com/landoop/lenses-go/pkg/sql"
	"github.com/landoop/lenses-go/pkg/topic"
	"github.com/landoop/lenses-go/pkg/user"
	"github.com/spf13/cobra"
)

var (
	app = &bite.Application{
		Name:        "lenses-cli",
		Description: "Lenses-cli is the command line client for the Lenses REST API.",
		Version:     "blop",
		ShowSpinner: false,
		Setup:       setup,
	}

	// buildRevision is the build revision (docker commit string or git rev-parse HEAD) but it's
	// available only on the build state, on the cli executable - via the "--version" flag.
	buildRevision = ""
	// buildTime is the build unix time (in seconds since 1970-01-01 00:00:00 UTC), like the `buildRevision`,
	// this is available on after the build state, inside the cli executable - via the "--version" flag.
	//
	// Note that this buildTime is not int64, it's type of string and it is provided at build time.
	// Do not change!
	buildTime    = ""
	buildVersion = ""
)

func setup(cmd *cobra.Command, args []string) error {
	ok, err := config.Manager.Load()
	// if command is "configure" and the configuration is invalid at this point, don't give a failure,
	// let the configure command give a tutorial for user in order to create a configuration file.
	// Note that if clientConfig is valid and we are inside the configure command
	// then the configure will normally continue and save the valid configuration (that normally came from flags).
	topLevelSubCmd := strings.Split(cmd.CommandPath(), " ")[1]
	if name := topLevelSubCmd; name == "configure" || name == "version" || name == "context" || name == "contexts" || strings.Contains(cmd.CommandPath(), " secrets ") {
		return nil
	}

	// it's not nil, if context does not exist then it would throw an error.
	currentConfig := config.Manager.Config.GetCurrent()
	for !ok {
		if err != nil {
			return err
		}

		if currentConfig.Debug {
			fmt.Fprintf(cmd.OutOrStdout(), "%#+v\n", *currentConfig)
		}

		fmt.Fprintln(cmd.OutOrStderr(), "cannot retrieve credentials, please configure below")
		configureCmd := user.NewConfigureCommand("")
		// disable any flags passed on the parent command before execute.
		configureCmd.DisableFlagParsing = true
		if err = configureCmd.Execute(); err != nil {
			return err
		}

		ok, err = config.Manager.Load()
	}

	// if login, remove the token so setupClient will generate a new one and save it to the home dir/lenses-cli.yml.
	if cmd.Name() == "login" {
		currentConfig.Token = ""

		if basicAuth, isBasicAuth := currentConfig.Authentication.(api.BasicAuthentication); isBasicAuth {
			//  and fire any errors if host or user or pass are not there.
			if currentConfig.Host == "" || basicAuth.Username == "" || basicAuth.Password == "" {
				// return fmt.Errorf("cannot retrieve credentials, please setup the configuration using the '%s' command first", "configure")
				//
				if err := user.NewConfigureCommand("").Execute(); err != nil {
					return err
				}

				// add a new line, so the login's session welcome messages has its place.
				fmt.Fprintln(cmd.OutOrStdout())
			}
		}

		return nil
	}

	return config.SetupClient()
}

func main() {

	if buildRevision != "" {
		app.HelpTemplate = bite.HelpTemplate{
			Name:                 "lenses-cli",
			BuildRevision:        buildRevision,
			BuildTime:            buildTime,
			BuildVersion:         buildVersion,
			ShowGoRuntimeVersion: true,
		}
	}

	if len(os.Args) == 1 || (string(os.Args[1]) != "secrets" && string(os.Args[1]) != "version") {
		app.PersistentFlags = config.SetupConfigManager
	} else {
		config.Manager = config.NewEmptyConfigManager()
		app.DisableOutputFormatController = true
	}

	//ACL
	app.AddCommand(acl.NewGetACLsCommand())
	app.AddCommand(acl.NewACLGroupCommand())

	//Alert
	app.AddCommand(alert.NewAlertGroupCommand())
	app.AddCommand(alert.NewGetAlertsCommand())

	//Audit
	app.AddCommand(audit.NewGetAuditEntriesCommand())

	//Config
	app.AddCommand(config.NewGetConfigsCommand())
	app.AddCommand(config.NewGetModeCommand())

	//Connectors
	app.AddCommand(connector.NewConnectorsCommand())
	app.AddCommand(connector.NewConnectorGroupCommand())

	//Export
	app.AddCommand(export.NewExportGroupCommand())

	//Import
	app.AddCommand(imports.NewImportGroupCommand())

	//Logs
	app.AddCommand(logs.NewLogsCommandGroup())

	//Policies
	app.AddCommand(policy.NewGetPoliciesCommand())
	app.AddCommand(policy.NewPolicyGroupCommand())

	//Processors
	app.AddCommand(processor.NewGetProcessorsCommand())
	app.AddCommand(processor.NewProcessorGroupCommand())

	//Topics
	app.AddCommand(topic.NewTopicsGroupCommand())
	app.AddCommand(topic.NewTopicGroupCommand())

	//Quotas
	app.AddCommand(quota.NewGetQuotasCommand())
	app.AddCommand(quota.NewQuotaGroupCommand())

	//Schemas
	app.AddCommand(schema.NewSchemasGroupCommand())
	app.AddCommand(schema.NewSchemaGroupCommand())

	//Shell
	app.AddCommand(shell.NewInteractiveCommand())

	//Secrets
	app.AddCommand(secret.NewSecretsGroupCommand())

	//SQL
	app.AddCommand(sql.NewLiveLSQLCommand())

	//User
	app.AddCommand(user.NewGetConfigurationContextsCommand())
	app.AddCommand(user.NewConfigurationContextCommand())
	app.AddCommand(user.NewConfigureCommand(""))
	app.AddCommand(user.NewLoginCommand(app))
	app.AddCommand(user.NewGetLicenseInfoCommand())
	app.AddCommand(user.NewUserGroupCommand())

	//Management
	app.AddCommand(management.NewGroupsCommand())
	app.AddCommand(management.NewUsersCommand())
	app.AddCommand(management.NewServiceAccountsCommand())

	if err := app.Run(os.Stdout, os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
