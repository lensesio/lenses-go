package conntemplate

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/test"
	"github.com/stretchr/testify/assert"
)

const connectionTemplateListResponse = `
[
  {
    "name": "JDBC",
    "version": "1.0.0",
    "enabled": true,
    "builtIn": true,
    "category": "Connection",
    "type": "Kafka Connect Source",
    "metadata": {
      "author": "Lenses.io",
      "description": "JDBC Connector as a Source",
      "applicationLanguage": {
        "name": "Not Applicable"
      }
    },
    "configuration": [
      {
        "key": "user",
        "displayName": "Username",
        "placeholder": "Username",
        "description": "Name for the connection",
        "type": {
          "name": "String",
          "displayName": "String"
        },
        "required": true,
        "mounted": false
      },
      {
        "key": "password",
        "displayName": "Password",
        "description": "Password to connect to the JDBC service",
        "type": {
          "name": "Secret",
          "displayName": "Secret"
        },
        "required": true,
        "mounted": false
      },
      {
        "key": "host",
        "displayName": "Hostname",
        "placeholder": "127.0.0.1",
        "description": "Host to connect to the JDBC service",
        "type": {
          "name": "String",
          "displayName": "String"
        },
        "required": true,
        "mounted": false
      },
      {
        "key": "driver",
        "displayName": "JDBC Driver",
        "placeholder": "jdbc.connector.Driver",
        "description": "Driver to use for connecting to the JDBC service",
        "type": {
          "name": "String",
          "displayName": "String"
        },
        "required": true,
        "mounted": false
      },
      {
        "key": "optional",
        "displayName": "Optional Field",
        "description": "Optional field for testing",
        "type": {
          "name": "String",
          "displayName": "String"
        },
        "required": false,
        "mounted": false
      },
      {
        "key": "optional array",
        "displayName": "Optional Array",
        "description": "Optional array for testing",
        "type": {
          "name": "Array",
          "displayName": "List"
        },
        "required": false,
        "mounted": false
      },
      {
        "key": "optional-boolean",
        "displayName": "Optional Boolean",
        "description": "Optional boolean for testing",
        "type": {
          "name": "Boolean",
          "displayName": "Boolean"
        },
        "required": false,
        "mounted": false
      },
      {
        "key": "optional-number",
        "displayName": "Optional Number",
        "description": "Optional number for testing",
        "type": {
          "name": "Number",
          "displayName": "Number"
        },
        "required": false,
        "mounted": false
      }
    ]
  }
]
`

func TestConnectionTemplateGroupCommandSuccess(t *testing.T) {
	// setup http request handler
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(connectionTemplateListResponse))
	})
	// setup http client
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	// test `connection-templates` command
	cmd := NewConnectionTemplateGroupCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var connections []api.ConnectionTemplate
	err = json.Unmarshal([]byte(output), &connections)

	assert.Nil(t, err)

	assert.Equal(t, "JDBC", connections[0].Name)
	config.Client = nil
}
