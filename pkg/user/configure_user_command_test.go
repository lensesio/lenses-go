package user

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/lensesio/lenses-go/pkg/api"
	test "github.com/lensesio/lenses-go/test"
)

const contextOutput = "[master] [valid, current]\n{\n  \"host\": \"http://domain.com:80\",\n  \"token\": \"****\",\n  \"timeout\": \"15s\",\n  \"debug\": true,\n  \"basic\": {\n    \"username\": \"user\",\n    \"password\": \"****\"\n  }\n}\n"

func TestContextCommands(t *testing.T) {
	scenarios := make(map[string]test.CommandTest)

	scenarios["Command 'context' should return the 'master' context when 'master' context exists"] =
		test.CommandTest{
			Setup:    test.SetupMasterContext,
			Teardown: test.ResetConfigManager,
			Cmd:      NewConfigurationContextCommand,
			ProcessOutput: func(t *testing.T, output string) {
				assert.Equal(t, output, contextOutput)
			},
			ShouldContain: []string{
				"master",
				"http://domain.com:80",
				"[master] [valid, current]",
				contextOutput,
			},
			ShouldNotContain: []string{
				"second",
				"http://example.com:80",
			},
		}

	scenarios["Command 'context' should return err when no context exists"] =
		test.CommandTest{
			Setup:    test.SetupConfigManager,
			Teardown: test.ResetConfigManager,
			Cmd:      NewConfigurationContextCommand,
			ShouldContainErrors: []string{
				"current context does not exist, please use the `configure` command first",
			},
			ShouldNotContain: []string{
				"master",
				"http://domain.com:80",
			},
		}

	scenarios["Command 'contexts' should return 'master' and 'second' contexts when both exists"] =
		test.CommandTest{
			Setup: func() {
				secondAuth := api.BasicAuthentication{
					Username: "user",
					Password: "pass",
				}
				secondClientConfig := api.ClientConfig{
					Authentication: secondAuth,
					Debug:          false,
					Host:           "example.com",
					Timeout:        "30s",
					Token:          "secret",
				}
				test.SetupContext("second", secondClientConfig, secondAuth)
				test.SetupMasterContext()
			},
			Teardown: test.ResetConfigManager,
			Cmd:      NewGetConfigurationContextsCommand,
			ShouldContain: []string{
				"master",
				"http://domain.com:80",
				"second",
				"http://example.com:80",
				contextOutput,
			},
			ShouldNotContain: []string{
				"third",
				"http://foo.com:80",
			},
		}

	scenarios["Command 'contexts' should return no contexts when none exists"] =
		test.CommandTest{
			Setup:    test.SetupConfigManager,
			Teardown: test.ResetConfigManager,
			Cmd:      NewGetConfigurationContextsCommand,
			ShouldNotContain: []string{
				"master",
				"http://domain.com:80",
				"second",
				"http://example.com:80",
				"third",
				"http://foo.com:80",
			},
		}

	test.RunCommandTests(t, scenarios)
}
