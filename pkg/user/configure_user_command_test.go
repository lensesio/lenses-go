package user

import (
	"testing"

	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	test "github.com/landoop/lenses-go/test"
	"github.com/stretchr/testify/assert"
)

const contextOutput = "[master] [valid, current]\n{\n  \"host\": \"http://domain.com:80\",\n  \"token\": \"****\",\n  \"timeout\": \"15s\",\n  \"debug\": true,\n  \"basic\": {\n    \"username\": \"user\",\n    \"password\": \"****\"\n  }\n}\n"

func TestContextCommandSuccess(t *testing.T) {
	test.SetupMasterContext()

	//test `context` cmd
	cmd := NewConfigurationContextCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.Equal(t, output, contextOutput)

	config.Client = nil
}

func TestSelectedContextCommandSuccess(t *testing.T) {
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
	test.SetupMasterContext()
	test.SetupContext("second", secondClientConfig, secondAuth)

	//test `contexts` cmd
	cmd := NewGetConfigurationContextsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.Contains(t, output, "master")
	assert.Contains(t, output, "http://domain.com:80")
	assert.Contains(t, output, "second")
	assert.Contains(t, output, "http://example.com:80")

	config.Client = nil
}
