package user

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/kataras/golog"

	"github.com/AlecAivazis/survey/v2"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

//NewGetConfigurationContextsCommand creates `contexts` command
func NewGetConfigurationContextsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "contexts",
		Short:         "Print and validate (through calls to the servers) all the available contexts from the configuration file",
		Example:       "contexts",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			for name := range config.Manager.Config.Contexts {
				if !printConfigurationContext(cmd, name) {
					if !bite.GetSilentFlag(cmd) {
						showOptionsForConfigurationContext(cmd, name)
					}
				}
			}
			return nil
		},
	}

	bite.CanBeSilent(cmd)

	return cmd
}

//NewConfigurationContextCommand creates `context` command
func NewConfigurationContextCommand() *cobra.Command {
	root := &cobra.Command{
		Use:           "context",
		Short:         "Print the current context or modify or delete a configuration context using the update and delete subcommands",
		Example:       `context`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// normally the cli would throw "client: credentials missing or invalid" if the current context's configuration
			// are invalid, but in the case of "context" command, we skip that setup on the root command.
			if !config.Manager.Config.CurrentContextExists() {
				return fmt.Errorf("current context does not exist, please use the `configure` command first")
			}
			name := config.Manager.Config.CurrentContext
			if !printConfigurationContext(cmd, name) {
				if !bite.GetSilentFlag(cmd) {
					showOptionsForConfigurationContext(cmd, name)
				}
			}
			return nil
		},
	}

	bite.CanBeSilent(root)

	root.AddCommand(NewUpdateConfigurationContextCommand())
	root.AddCommand(NewDeleteConfigurationContextCommand())
	root.AddCommand(NewUseContextCommand())

	return root
}

//NewDeleteConfigurationContextCommand creates `context delete` command
func NewDeleteConfigurationContextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "delete",
		Short:         "Delete a configuration context",
		Example:       `context delete context_name`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("one argument is required for the context name")
			}

			name := args[0]
			removeContextWillChangeContext := config.Manager.Config.CurrentContext == name
			deleted := config.Manager.Config.RemoveContext(name)

			if !deleted {
				// failed when no found this context or if we can't upgrade to another one.
				return fmt.Errorf("unable to delete context [%s], at least one more valid context should be present", name)
			}

			if err := config.Manager.Save(); err != nil {
				return fmt.Errorf("error while saving the configuration after deletion of the [%s] context: [%v]", name, err)
			}

			succMsg := fmt.Sprintf("[%s] context deleted", name)

			if removeContextWillChangeContext {
				newCurrentContext := config.Manager.Config.CurrentContext
				succMsg = fmt.Sprintf("[%s], current context set to [%s]", succMsg, newCurrentContext)
			}

			return bite.PrintInfo(cmd, succMsg)

		},
	}

	bite.CanBeSilent(cmd)

	return cmd
}

//NewUpdateConfigurationContextCommand creates `context set` command
func NewUpdateConfigurationContextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "set",
		Aliases:       []string{"edit", "update", "create", "add"},
		Short:         "Edit an existing or add a configuration context e.g. lenses-cli context create my-new-context",
		Example:       `context edit context_name`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("one argument is required for the context name")
			}

			name := args[0]

			configureCmd := NewConfigureCommand(name)
			configureCmd.Flag("reset").Value.Set("true")
			// these wil disable banner and location save, note that if --file is there then it will take that, otherwise the default $HOME/.lenses/lenses-cli.yml.
			configureCmd.Flag("no-banner").Value.Set("true")
			configureCmd.Flag("default-location").Value.Set("true")
			if err := configureCmd.Execute(); err != nil {
				return err
			}

			if isValidConfigurationContext(name) {
				return bite.PrintInfo(cmd, "[%s] was successfully validated and saved, it is the current context now", name)
			}

			retry := true
			if err := survey.AskOne(&survey.Confirm{
				Message: fmt.Sprintf("[%s] is invalid, connection failed, do you mind to retry fixing it?", name),
				Default: true,
			}, &retry, nil); err != nil {
				return err
			}

			if retry {
				newCmd := NewUpdateConfigurationContextCommand()
				newCmd.SetArgs(args)
				return newCmd.Execute()
			}

			return nil
		},
	}

	return cmd
}

//NewUseContextCommand creates `context use` command
func NewUseContextCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:           "use",
		Short:         "use a context",
		Example:       `context use context_name`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("one argument is required for the context name")
			}

			name := args[0]

			if config.Manager.Config.ContextExists(name) {
				config.Manager.Config.SetCurrent(name)
				config.Manager.Save()
				bite.PrintInfo(cmd, "Current context set to [%s]", name)
				return nil
			}

			golog.Errorf("Context [%s] not found", name)
			return nil
		},
	}

	return cmd
}

//NewConfigureCommand creates `configure` command
func NewConfigureCommand(name string) *cobra.Command {
	var (
		reset       bool
		noBanner    bool // if true doesn't print the banner (useful for running inside other commands).
		defLocation bool // if true doesn't asks for location to save (useful for running inside other commands).
	)

	cmd := &cobra.Command{
		Use:           "configure",
		Short:         "Setup your environment for extensive CLI use. Create and save the required CLI configuration and client credentials",
		Example:       `configure`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !config.Manager.Config.IsValid() || reset {
				// This is the only command and place the user has direct interaction with the CLI
				// and it's not used by a third-party tool because of the survey.
				// So, print our "banner" :)
				if !noBanner {
					fmt.Fprint(cmd.OutOrStdout(), `
    __                                 ________    ____
   / /   ___  ____  ________  _____   / ____/ /   /  _/
  / /   / _ \/ __ \/ ___/ _ \/ ___/  / /   / /    / /
 / /___/  __/ / / (__  )  __(__  )  / /___/ /____/ /
/_____/\___/_/ /_/____/\___/____/   \____/_____/___/
Docs at https://docs.lenses.io
`)
				}

				// if the current is not the specified one set it to the new
				config.Manager.Config.SetCurrent(name)
				currentConfig := config.Manager.Config.GetCurrent()

				var (
					defUsername  string
					defKrbFile   string
					defKrbRealm  string
					defKrbKeytab string
					defKrbCCache string
				)

				switch auth := currentConfig.Authentication.(type) {
				case api.BasicAuthentication:
					defUsername = auth.Username
				case api.KerberosAuthentication:
					defKrbFile = auth.ConfFile

					switch authMethod := auth.Method.(type) {
					case api.KerberosWithPassword:
						defUsername = authMethod.Username
						defKrbRealm = authMethod.Realm
					case api.KerberosWithKeytab:
						defUsername = authMethod.Username
						defKrbRealm = authMethod.Realm
						defKrbKeytab = authMethod.KeytabFile
					case api.KerberosFromCCache:
						defKrbCCache = authMethod.CCacheFile
					}
				}

				qs := []*survey.Question{
					{
						Name: "debug",
						Prompt: &survey.Confirm{
							Message: "Enable debug mode?",
							Default: currentConfig.Debug,
						},
					},
					{
						Name: "insecure",
						Prompt: &survey.Confirm{
							Help:    "If you answer yes, the server's certificate will not be checked for validity. This will make your HTTPS connections insecure.",
							Message: "Enable insecure https connections?",
							Default: currentConfig.Insecure,
						},
					},
					{
						Name: "host",
						Prompt: &survey.Input{
							Message: "Host",
							Default: currentConfig.Host,
							Help:    "This is your lenses box host full address, including the schema and the port. The address that this Client will be connected to.",
						},
						Validate: survey.Required,
					},
				}

				if err := survey.Ask(qs, currentConfig); err != nil {
					return err
				}

				var (
					basicAuthAns    = "lenses BASIC auth or LDAP (default)"
					kerberosAuthAns = "kerberos (three methods)"
				)

				var authAns string

				if err := survey.AskOne(&survey.Select{
					Message: fmt.Sprintf("How would you like to be authenticated?"),
					Options: []string{basicAuthAns, kerberosAuthAns},
				}, &authAns, nil); err != nil {
					return err
				}

				switch authAns {
				case kerberosAuthAns:
					var kerberosAuth api.KerberosAuthentication

					// get the krb5 conf file for all of the kerberos methods and ask for a method.
					if err := survey.AskOne(&survey.Input{
						Message: "krb5.conf file location",
						Default: defKrbFile,
						Help:    "This is the local kerberos configuration file.",
					}, &kerberosAuth.ConfFile, survey.WithValidator(survey.Required)); err != nil {
						return err
					}

					var (
						kerberosWithPassAns   = "kerberos with password"
						kerberosWithKeytabAns = "kerberos with keytab file"
						kerberosFromCCacheAns = "kerberos from ccache file"
					)

					var authMethodAns string

					if err := survey.AskOne(&survey.Select{
						Message: fmt.Sprintf("Please select one of the following kerberos authentication methods"),
						Options: []string{kerberosWithPassAns, kerberosWithKeytabAns, kerberosFromCCacheAns},
					}, &authMethodAns, nil); err != nil {
						return err
					}

					switch authMethodAns {
					case kerberosWithPassAns:

						qs = []*survey.Question{
							{
								Name: "realm",
								Prompt: &survey.Input{
									Message: "Realm",
									Default: defKrbRealm,
									Help:    "This is the realm, if empty then the default realm will be used.",
								},
							},
							{
								Name: "username",
								Prompt: &survey.Input{
									Message: "Username",
									Default: defUsername,
									Help:    "This is the user credential used for gain access to the API.",
								},
								Validate: survey.Required,
							},
							{
								Name: "password",
								Prompt: &survey.Password{
									Message: "Password",
									Help:    "This is the user's password credential, necessary to gain access to the API.",
								},
								Validate: survey.Required,
							},
						}

						var kerberosMethod api.KerberosWithPassword
						if err := survey.Ask(qs, &kerberosMethod); err != nil {
							return err
						}

						kerberosAuth.Method = kerberosMethod
					case kerberosWithKeytabAns:

						qs = []*survey.Question{
							{
								Name: "realm",
								Prompt: &survey.Input{
									Message: "Realm",
									Default: defKrbRealm,
									Help:    "This is the realm, if empty then the default realm will be used.",
								},
							},
							{
								Name: "username",
								Prompt: &survey.Input{
									Message: "Username",
									Default: defUsername,
									Help:    "This is the user credential used for gain access to the API.",
								},
								Validate: survey.Required,
							},
							{
								Name: "keytab",
								Prompt: &survey.Input{
									Message: "Keytab file location",
									Default: defKrbKeytab,
									Help:    "This is the local generated keytab file location.",
								},
							},
						}

						var kerberosMethod api.KerberosWithKeytab
						if err := survey.Ask(qs, &kerberosMethod); err != nil {
							return err
						}

						kerberosAuth.Method = kerberosMethod
					case kerberosFromCCacheAns:
						qs = []*survey.Question{
							{
								Name: "ccache",
								Prompt: &survey.Input{
									Message: "CCache file location",
									Default: defKrbCCache,
									Help:    "This is the local ccache file location.",
								},
							},
						}

						var kerberosMethod api.KerberosFromCCache
						if err := survey.Ask(qs, &kerberosMethod); err != nil {
							return err
						}

						kerberosAuth.Method = kerberosMethod
					default:
						return fmt.Errorf("what?")
					}

					currentConfig.Authentication = kerberosAuth

				default:
					// basic auth.
					qs = []*survey.Question{
						{
							Name: "username",
							Prompt: &survey.Input{
								Message: "Username",
								Default: defUsername,
								Help:    "This is the user credential used for gain access to the API.",
							},
							Validate: survey.Required,
						},
						{
							Name: "password",
							Prompt: &survey.Password{
								Message: "Password",
								Help:    "This is the user's password credential, necessary to gain access to the API.",
							},
							Validate: survey.Required,
						},
					}

					var basicAuth api.BasicAuthentication
					if err := survey.Ask(qs, &basicAuth); err != nil {
						return err
					}

					currentConfig.Authentication = basicAuth
				}
				//
				// If all ok continue by saving the result to the desired system filepath.
				//

				// if already saved once and want to add more contexts, then don't ask for system path.
				if config.Manager.Config.CurrentContext != "" && len(config.Manager.Config.Contexts) > 0 {
					defLocation = true
				}

				if config.Manager.Filepath == "" && !defLocation { // if no --config is provided then ask.
					if err := survey.AskOne(&survey.Input{
						Message: "Save configuration file to",
						Default: config.DefaultConfigFilepath,
						Help:    "This is the system filepath to save the configuration which includes the credentials",
					}, &config.Manager.Filepath, nil); err != nil {
						return err
					}
				}

			} else {
				nFlags := cmd.Root().Flags().NFlag()
				if nFlags == 0 || (nFlags == 1 && cmd.Root().Flag("context").Changed) || (nFlags <= 2 && cmd.Root().Flag("config").Changed) {
					// flags given like --user and --pass and --host, then we don't want to save anything,
					// user may need to re-configure, give a note about the --reset flag.
					return fmt.Errorf("configuration already exists, try 'configure --reset' instead")
				}
			}

			return config.Manager.Save()
		},
	}
	cmd.Flags().BoolVar(&reset, "reset", false, "reset the current configuration")
	cmd.Flags().BoolVar(&noBanner, "no-banner", false, "disables the banner output")
	cmd.Flags().BoolVar(&defLocation, "default-location", false, "will not ask for the location to save on, the result will be saved to the $HOME/.lenses/lenses-cli.yml")
	return cmd
}

//NewLoginCommand create `login` command
func NewLoginCommand(app *bite.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "login",
		Short:            "Login, generate the access token using the generated configuration via the 'configure' command. ",
		Example:          `login`,
		SilenceErrors:    true,
		TraverseChildren: true,
		Hidden:           true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			if _, err := api.OpenConnection(*config.Manager.Config.GetCurrent()); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			signedUser := client.User
			fmt.Fprintf(out, "Welcome [%s%s],\ntype 'help' to learn more about the available commands or 'exit' to terminate.\n",
				signedUser.Name, strings.Join(signedUser.Permissions, ", "))

			// read the input pipe, on each read its buffer consumed, so loop 'forever' here.
			streamReader := bufio.NewReader(os.Stdin)
			for {

				// add the 'ready to type' signal for the user.

				fmt.Fprint(out, "> ")

				line, err := streamReader.ReadString('\n')
				if err != nil {
					return err // exit on first failure.
				}

				// remove any last \r\n.
				line = strings.TrimRight(line, "\r\n")

				// if "exit" then exit now.
				switch line {
				case "exit":
					os.Exit(0)
				case "clear", "cls":
					if runtime.GOOS == "windows" {
						// TODO: not tested yet.
						cmd := exec.Command("cmd", "/c", "cls")
						cmd.Stdout = out
						cmd.Run()
					} else {
						cmd.Print("\033[H\033[2J")
					}

					continue
				}

				cms := strings.Split(line, " ")

				// parse the line (as slice of strings) in order to take the command and the flags from it.

				cP, flags := app.FindCommand(cms)
				if cP == nil {
					fmt.Fprintln(out, fmt.Sprintf("command form of [%s] not found", line))
					continue
				}

				c := *cP
				commandName := c.Name()

				// check if "login" or "configure" commands, these cannot be run in the terminal session
				// for obvious reasons.
				if commandName == "login" || commandName == "configure" {
					fmt.Fprintln(out, "unable to run inside a started session")
					continue
				}

				// parse the flags found by the `Find`.
				if err = c.ParseFlags(flags); err != nil {
					fmt.Fprintln(out, err)
					continue
				}

				// see if we have arguments to set, arguments come after the flags.
				var cArgs []string
				if stopFlags := len(flags) + 1; len(cms) > stopFlags {
					cArgs = cms[1:stopFlags]
				}

				// run the command.
				c.DisableFlagParsing = true
				c.DisableFlagsInUseLine = true
				c.SetArgs(cArgs)

				if c.Run == nil && c.RunE == nil {
					// propably report as bug if this will happen ever.
					fmt.Fprintln(out, "command is unable to run")
					continue
				}

				if c.Run != nil {
					c.Run(&c, cArgs)
				} else if err = c.RunE(&c, cArgs); err != nil {
					fmt.Fprintln(out, err)
					// don't break this yet, let it to print an extra line if it was caused by the child command itself,
					// also the "logout" command can check for that error as well.
				}
				// a new line on succeed operations.
				fmt.Fprintln(out)
			}

		}}

	return cmd
}

func isValidConfigurationContext(name string) bool {
	currentContext := config.Manager.Config.CurrentContext
	config.Manager.Config.SetCurrent(name)
	_, err := api.OpenConnection(*config.Manager.Config.GetCurrent())
	if err != nil {
		return false
	}
	config.Manager.Config.SetCurrent(currentContext)
	return true
}

func printConfigurationContext(cmd *cobra.Command, name string) bool {
	currentContextName := config.Manager.Config.CurrentContext
	if len(config.Manager.Config.Contexts) == 0 {
		return false
	}

	c, ok := config.Manager.Config.Contexts[name]
	if !ok {
		return false // this should never happen.
	}

	c.FormatHost()
	cfg := *c
	if cfg.Token != "" {
		cfg.Token = "****"
	}

	// remove any password-based literals from the printable client config.
	if auth, ok := cfg.IsBasicAuth(); ok {
		auth.Password = "****"
		cfg.Authentication = auth
	} else if authKerb, ok := cfg.IsKerberosAuth(); ok {
		if authMethod, ok := authKerb.WithPassword(); ok {
			authMethod.Password = "****"
			authKerb.Method = authMethod
			cfg.Authentication = authKerb
		}
	}

	isValid := isValidConfigurationContext(name)
	info := "valid"
	if !isValid {
		info = "invalid"
	}
	b, err := api.ClientConfigMarshalJSON(cfg)
	if err != nil {
		isValid = false
		// these type of error reporting is not for end-user specific language
		// but they may help us on debugging if user edited manually the configs and was wrong.
		info += ", error: " + err.Error()
	}

	buf := new(bytes.Buffer)
	if err = json.Indent(buf, b, "", "  "); err != nil {
		if isValid {
			info += " but"
		} else {
			info += " and"
		}

		info += " unable to indent"

		isValid = false
	}

	if name == currentContextName {
		info += ", current"
	}

	fmt.Fprintf(cmd.OutOrStdout(), "[%s] [%s]\n", name, info)

	// buf.WriteTo(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), buf.String())
	// show only filled but no authentication.
	// bite.PrintJSON(cmd, cfg) // keep json?

	return isValid
}

func showOptionsForConfigurationContext(cmd *cobra.Command, name string) error {
	var action string

	if err := survey.AskOne(&survey.Select{
		Message: fmt.Sprintf("Would you like to skip, edit or delete the [%s] invalid configuration context?", name),
		Options: []string{"skip", "edit", "delete"},
	}, &action, nil); err != nil {
		return err
	}

	if action == "skip" {
		return nil
	}

	if action == "delete" {
		deleteCmd := NewDeleteConfigurationContextCommand()
		return deleteCmd.RunE(deleteCmd, []string{name})
	}

	if action == "edit" {
		editCmd := NewUpdateConfigurationContextCommand()
		editCmd.SetArgs([]string{name})
		if err := editCmd.Execute(); err != nil {
			return err
		}
	}

	return nil
}
