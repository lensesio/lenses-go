package main

import (
	"bufio"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/kataras/survey"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

func init() {
	rootCmd.AddCommand(newConfigureCommand())
	rootCmd.AddCommand(newLoginCommand())
	rootCmd.AddCommand(newGetUserInfoCommand())
	// remove `logout` command (at least for the moment) rootCmd.AddCommand(newLogoutCommand())
}

// Note that configure will never be called if home configuration is already exists, even if `lenses-cli configure`,
// this is an expected behavior to prevent any actions by mistakes from the user.
func newConfigureCommand() *cobra.Command {
	var (
		reset bool
	)

	cmd := &cobra.Command{
		Use:           "configure",
		Short:         "Setup your environment for extensive CLI use. Create and save the required CLI configuration and client credentials",
		Example:       exampleString(`configure`),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !config.IsValid() || reset {
				// This is the only command and place the user has direct interaction with the CLI
				// and it's not used by a third-party tool because of the survey.
				// So, print our "banner" :)
				cmd.Println(`
 ___      _______  __    _  _______  _______  _______ 
|   |    |       ||  |  | ||       ||       ||       |
|   |    |    ___||   |_| ||  _____||    ___||  _____|
|   |    |   |___ |       || |_____ |   |___ | |_____ 
|   |___ |    ___||  _    ||_____  ||    ___||_____  |
|       ||   |___ | | |   | _____| ||   |___  _____| |
|_______||_______||_|  |__||_______||_______||_______|
`)
				// flags are missing, so we don't have something to save directly,
				// ask the user to complete the neseccary (or missing from flags, yes host may given but user no for example)
				// fields with question-answer (prompt) system.
				// TODO: prompt.
				qs := []*survey.Question{
					{
						Name: "host",
						Prompt: &survey.Input{
							Message: "Host",
							Default: config.Host,
							Help:    "This is your lenses box host full address, including the schema and the port. The address that this Client will be connected to.",
						},
						Validate: survey.Required,
					},
					{
						Name: "user",
						Prompt: &survey.Input{
							Message: "User",
							Default: config.User,
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
					{
						Name: "debug",
						Prompt: &survey.Confirm{
							Message: "Enable debug mode?",
							Default: config.Debug,
						},
					},
				}

				if err := survey.Ask(qs, &config); err != nil {
					return err
				} // else continue by saving the result to the desired system filepath.

				if configFilepath == "" { // if no --config is provided then ask.
					if err := survey.AskOne(&survey.Input{
						Message: "Save configuration file to",
						Default: defaultConfigFilepath,
						Help:    "This is the system filepath to save the configuration which includes the credentials",
					}, &configFilepath, nil); err != nil {
						return err
					}
				}
			} else if cmd.Root().Flags().NFlag() == 0 {
				// flags given like --user and --pass and --host, then we don't want to save anything,
				// user may need to re-configure, give a note about the --reset flag.
				return fmt.Errorf("configuration already exists, try 'configure --reset' instead")
			}

			return saveConfiguration()
		},
	}

	cmd.Flags().BoolVar(&reset, "reset", false, "reset the current configuration")
	return cmd
}

func toHash(plain string) []byte {
	h := sha256.Sum256([]byte(plain))
	return h[:]
}

func encryptAES(key, data []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	out := make([]byte, aes.BlockSize+len(data))
	iv := out[:aes.BlockSize]
	encrypted := out[aes.BlockSize:]

	if _, err = io.ReadFull(rand.Reader, iv); err != nil {
		return nil, err
	}

	stream := cipher.NewCFBEncrypter(block, iv)
	stream.XORKeyStream(encrypted, data)

	return out, nil
}

func decryptAES(key, h []byte) ([]byte, error) {
	iv := h[:aes.BlockSize]
	h = h[aes.BlockSize:]

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	str := cipher.NewCFBDecrypter(block, iv)
	str.XORKeyStream(h, h)

	return h, nil
}

func encryptString(plain string, keyBase string) (string, error) {
	key := toHash(keyBase)
	encrypted, err := encryptAES(key, []byte(plain))
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(encrypted), nil
}

func decryptString(encryptedRaw string, keyBase string) (plainTextString string, err error) {
	encrypted, err := base64.URLEncoding.DecodeString(encryptedRaw)
	if err != nil {
		return "", err
	}

	if len(encrypted) < aes.BlockSize {
		return "", fmt.Errorf("short cipher, min len: 16")
	}

	decrypted, err := decryptAES(toHash(keyBase), encrypted)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

var defaultConfigFilepath = filepath.Join(lenses.DefaultConfigurationHomeDir, "lenses-cli.yml")

func saveConfiguration() error {
	p, err := encryptString(config.Password, config.Host)
	if err != nil {
		return err
	}

	config.Password = p

	out, err := yaml.Marshal(config)
	if err != nil { // should never happen.
		return fmt.Errorf("unable to marshal the configuration, error: %v", err)
	}

	if configFilepath == "" {
		configFilepath = defaultConfigFilepath
	}

	directoryMode := os.FileMode(0750)
	// create any necessary directories.
	os.MkdirAll(filepath.Dir(configFilepath), directoryMode)

	config.Token = "" // remove token

	fileMode := os.FileMode(0600)
	// if file exists it overrides it.
	if err = ioutil.WriteFile(configFilepath, out, fileMode); err != nil {
		return fmt.Errorf("unable to create the configuration file for your system, error: %v", err)
	}

	return nil
}

func newLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "login",
		Short:            "Login, generate the access token using the generated configuration via the 'configure' command. ",
		Example:          exampleString(`login`),
		SilenceErrors:    true,
		TraverseChildren: true,
		Hidden:           true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := setupClient(config.Configuration); err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			signedUser := client.User()
			fmt.Fprintf(out, "Welcome %s[%s],\ntype 'help' to learn more about the available commands or 'exit' to terminate.\n",
				signedUser.Name, strings.Join(signedUser.Roles, ", "))
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
				if line == "exit" {
					os.Exit(0)
				}

				cms := strings.Split(line, " ")

				/* Remember: why we do this "cP"?:
				   if not then:
				    processors --no-pretty
				    and after
				    processors
				    will keep the --no-pretty flag to true without be able to change it via --no-pretty=false.

				    With the clone solution we still remember the flags(very important) but they can be changed if needed.
				*/

				// parse the line (as slice of strings) in order to take the command and the flags from it.
				cP, flags, err := rootCmd.Find(cms)
				if err != nil {
					fmt.Fprintln(out, err)
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

				// if command was "logout" then exit.
				if commandName == logoutCmdName {
					if err != nil {
						os.Exit(1)
					}
					os.Exit(0)
				}

				// a new line on succeed operations.
				fmt.Fprintln(out)
			}

		}}

	return cmd
}

func newGetUserInfoCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "user",
		Short:            "Print some information about the authenticated logged user such as the given roles given by the lenses administrator",
		Example:          exampleString("user"),
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if user := client.User(); user.ID != "" {
				// if logged in using the user password, then we have those info,
				// let's print it as well.
				return printJSON(cmd, user)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")
	return cmd
}

const logoutCmdName = "logout"

// func newLogoutCommand() *cobra.Command {
// 	cmd := &cobra.Command{
// 		Use:              logoutCmdName,
// 		Short:            "Revoke the access token",
// 		Example:          exampleString(logoutCmdName),
// 		TraverseChildren: true,
// 		RunE: func(cmd *cobra.Command, args []string) error {
// 			if err := client.Logout(); err != nil {
// 				return err // first re-voke the access token.
// 			}

// 			// after remove the token from the configuration.
// 			config.Token = ""
// 			return saveConfiguration()
// 		},
// 	}

// 	return cmd
// }
