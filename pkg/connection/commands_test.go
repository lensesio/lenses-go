package connection

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/test"
	"github.com/stretchr/testify/assert"
)

const connectionListResponse = `
[
  {
    "name": "TestConn0",
    "templateName": "Slack",
    "templateVersion": 1,
    "tags": [
      "t1"
    ],
    "readOnly": false
  },
  {
    "name": "TestConn1",
    "templateName": "Slack",
    "templateVersion": 1,
    "tags": [
      "t2"
    ],
    "readOnly": true
  }
]
`

const connectionGetResponse = `
{
	"name": "TestConn0",
	"templateVersion": 1,
	"templateName": "Slack",
	"builtIn": true,
	"createdBy": "admin",
	"createdAt": 1580392100854,
	"modifiedBy": "admin",
	"modifiedAt": 1580392100854,
	"config": [
	  {
		"name": "webhookUrl",
		"value": "https://hooks.slack.com/",
		"type": "STRING",
		"mounted": false
	  }
	],
	"readOnly": false,
	"tags": [
	  "t1"
	]
}
`

func TestConnectionGroupCommandSuccess(t *testing.T) {
	// setup http request handler
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(connectionListResponse))
	})
	// setup http client
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	// test `connections` command
	cmd := NewConnectionGroupCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var connections []api.Connection
	err = json.Unmarshal([]byte(output), &connections)

	assert.Nil(t, err)

	assert.Equal(t, "TestConn0", connections[0].Name)
	config.Client = nil
}

func TestConnectionGetCommandSuccess(t *testing.T) {
	// setup http request handler
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(connectionGetResponse))
	})
	// setup http client
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	// test `connections get` command
	cmd := NewConnectionGetCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "--name=TestConn0")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var connection api.Connection
	err = json.Unmarshal([]byte(output), &connection)

	assert.Nil(t, err)

	assert.Equal(t, "TestConn0", connection.Name)
	config.Client = nil
}

func TestConnectionCreateCommandSuccess(t *testing.T) {
	// setup http request handler
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(connectionListResponse))
	})
	// setup http client
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	// test `connections create` command
	cmd := NewConnectionCreateCommand()
	output, err := test.ExecuteCommand(cmd, "--name=TestConnection",
		"--tag=t1",
		"--tag=t2",
		"--template-name=Slack",
		"--connection-config=[{\"key\":\"webhookUrl\",\"value\":\"https://hooks.slack.com/\"}]",
	)

	assert.Nil(t, err)
	assert.Equal(t, "Lenses connection has been successfully created.\n", output)

	config.Client = nil
}

func TestConnectionUpdateCommandSuccess(t *testing.T) {
	// setup http request handler
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(connectionListResponse))
	})
	// setup http client
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	// test `connections update` command
	cmd := NewConnectionUpdateCommand()
	output, err := test.ExecuteCommand(cmd, "--name=TestConnection",
		"--tag=t3",
		"--connection-config=[{\"key\":\"webhookUrl\",\"value\":\"https://hooks.slack.com/\"}]",
	)

	assert.Nil(t, err)
	assert.Equal(t, "Lenses connection has been successfully updated.\n", output)

	config.Client = nil
}

func TestConnectionDeleteCommandSuccess(t *testing.T) {
	// setup http request handler
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(connectionListResponse))
	})
	// setup http client
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	// test `connections delete` command
	cmd := NewConnectionDeleteCommand()
	output, err := test.ExecuteCommand(cmd, "--name=connection-name")

	assert.Nil(t, err)
	assert.Equal(t, "Lenses connection has been successfully deleted.\n", output)

	config.Client = nil
}
