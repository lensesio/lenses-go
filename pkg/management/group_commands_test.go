package management

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/test"
	"github.com/stretchr/testify/assert"
)

const groupsOkReponse = `[
  {
    "name": "DevAdmin",
    "description": "some info 2",
    "namespaces": [
      {
        "wildcards": [
          "*"
        ],
        "permissions": [
          "CreateTopic",
          "DropTopic",
          "ConfigureTopic",
          "QueryTopic",
          "ShowTopic",
          "ViewTopicMetadata"
        ],
        "system": "Kafka",
        "instance": "Dev"
      },
      {
        "wildcards": [
          "abc*"
        ],
        "permissions": [
          "QueryTopic",
          "ShowTopic",
          "ViewTopicMetadata"
        ],
        "system": "Kafka",
        "instance": "Dev"
      }
    ],
    "scopedPermissions": [
      "ViewKafkaConsumers",
      "ManageKafkaConsumers",
      "ViewConnectors",
      "ManageConnectors",
      "ViewSQLProcessors",
      "ManageSQLProcessors",
      "ViewCustomApps",
      "ManageCustomApps",
      "ViewSchemas",
      "ViewTopology",
      "ManageTopology"
    ],
    "adminPermissions": [
      "ViewDataPolicies",
      "ViewAuditLogs",
      "ViewUsers",
      "ManageUsers",
      "ViewAlertSettings",
      "ManageAlertSettings",
      "ViewKafkaSettings",
      "ManageKafkaSettings"
    ],
    "userAccounts": 1,
    "serviceAccounts": 0
  },
  {
    "name": "janitors",
    "description": "only head of janitors",
    "namespaces": [
      {
        "wildcards": [
          "*"
        ],
        "permissions": [
          "ConfigureTopic",
          "QueryTopic",
          "ShowTopic",
          "ViewTopicMetadata"
        ],
        "system": "Kafka",
        "instance": "Dev"
      }
    ],
    "scopedPermissions": [
      "ViewKafkaConsumers",
      "ManageKafkaConsumers",
      "ViewConnectors",
      "ManageConnectors",
      "ViewSQLProcessors",
      "ViewCustomApps",
      "ViewSchemas",
      "ViewTopology"
    ],
    "adminPermissions": [
      "ViewDataPolicies",
      "ViewAuditLogs",
      "ViewUsers",
      "ManageUsers",
      "ViewAlertSettings",
      "ManageAlertSettings",
      "ViewKafkaSettings",
      "ManageKafkaSettings"
    ],
    "userAccounts": 0,
    "serviceAccounts": 0
  }
]
`

func TestGroupsCommandSuccess(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(groupsOkReponse))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	//test `groups` cmd
	cmd := NewGroupsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	config.Client = nil
}

func TestGroupsCommandHttpFail(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	//test `poicies` cmd
	cmd := NewGroupsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	_, err = test.ExecuteCommand(cmd)

	assert.NotNil(t, err)

	config.Client = nil
}

const groupOkResponse = `{
	"name": "DevAdmin",
	"description": "some info 2",
	"namespaces": [
	  {
		"wildcards": [
		  "*"
		],
		"permissions": [
		  "CreateTopic",
		  "DropTopic",
		  "ConfigureTopic",
		  "QueryTopic",
		  "ShowTopic",
		  "ViewTopicMetadata"
		],
		"system": "Kafka",
		"instance": "Dev"
	  },
	  {
		"wildcards": [
		  "abc*"
		],
		"permissions": [
		  "QueryTopic",
		  "ShowTopic",
		  "ViewTopicMetadata"
		],
		"system": "Kafka",
		"instance": "Dev"
	  }
	],
	"scopedPermissions": [
	  "ViewKafkaConsumers",
	  "ManageKafkaConsumers",
	  "ViewConnectors",
	  "ManageConnectors",
	  "ViewSQLProcessors",
	  "ManageSQLProcessors",
	  "ViewCustomApps",
	  "ManageCustomApps",
	  "ViewSchemas",
	  "ViewTopology",
	  "ManageTopology"
	],
	"adminPermissions": [
	  "ViewDataPolicies",
	  "ViewAuditLogs",
	  "ViewUsers",
	  "ManageUsers",
	  "ViewAlertSettings",
	  "ManageAlertSettings",
	  "ViewKafkaSettings",
	  "ManageKafkaSettings"
	],
	"userAccounts": 1,
	"serviceAccounts": 0
  }`

func TestGroupViewCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(groupOkResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGroupsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "get", "--name=DevAdmin")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var group api.Group
	err = json.Unmarshal([]byte(output), &group)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "DevAdmin", group.Name)
	assert.Equal(t, 11, len(group.ScopedPermissions))
	assert.Equal(t, 8, len(group.AdminPermissions))

	config.Client = nil
}

func TestGroupCreateCommandFailMissingFields(t *testing.T) {
	cmd := NewGroupsCommand()
	_, err := test.ExecuteCommand(cmd, "create", "--name=MyGroup")

	assert.NotNil(t, err)

	config.Client = nil
}

func TestGroupCreateCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGroupsCommand()
	output, err := test.ExecuteCommand(cmd, "create",
		"--name=MyGroup",
		"--description=mygroup",
		"--applicationPermissions=ViewKafkaConsumers",
		"--applicationPermissions=ViewSQLProcessors",
		`--dataNamespaces`,
		`[{"wildcards":["*"],"permissions":["CreateTopic","DropTopic","ConfigureTopic","QueryTopic","ShowTopic","ViewTopicMetadata","InsertData","DeleteData","UpdateTablestore"],"system":"Kafka","instance":"Dev"}]`,
	)
	assert.Nil(t, err)
	assert.Equal(t, "Group [MyGroup] created\n", output)
	config.Client = nil
}

func TestGroupUpdateCommandFailMissingFields(t *testing.T) {

	cmd := NewGroupsCommand()
	_, err := test.ExecuteCommand(cmd, "update", "--name=MyGroup")

	assert.NotNil(t, err)

	config.Client = nil
}

func TestGroupUpdateCommandHttpFail(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGroupsCommand()
	_, err = test.ExecuteCommand(cmd, "update",
		"--name=MyGroup",
		"--description=mygroup",
		"--applicationPermissions=ViewKafkaConsumers",
		"--applicationPermissions=ViewSQLProcessors",
		`--dataNamespaces`,
		`[{"wildcards":["*"],"permissions":["CreateTopic","DropTopic","ConfigureTopic","QueryTopic","ShowTopic","ViewTopicMetadata","InsertData","DeleteData","UpdateTablestore"],"system":"Kafka","instance":"Dev"}]`,
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestGroupUpdateCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGroupsCommand()
	output, err := test.ExecuteCommand(cmd, "update",
		"--name=MyGroup",
		"--description=mygroup",
		"--applicationPermissions=ViewKafkaConsumers",
		"--applicationPermissions=ViewSQLProcessors",
		`--dataNamespaces`,
		`[{"wildcards":["*"],"permissions":["CreateTopic","DropTopic","ConfigureTopic","QueryTopic","ShowTopic","ViewTopicMetadata","InsertData","DeleteData","UpdateTablestore"],"system":"Kafka","instance":"Dev"}]`,
	)
	assert.Nil(t, err)
	assert.Equal(t, "Group [MyGroup] updated\n", output)
	config.Client = nil
}

func TestGroupDeleteMissingFieldsFails(t *testing.T) {
	cmd := NewGroupsCommand()
	_, err := test.ExecuteCommand(cmd, "delete",
		"--name=",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestGroupDeleteHttpFails(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGroupsCommand()
	_, err = test.ExecuteCommand(cmd, "delete",
		"--name=MyGroup",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestGroupDeleteSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGroupsCommand()
	output, err := test.ExecuteCommand(cmd, "delete",
		"--name=MyGroup",
	)
	assert.Nil(t, err)
	assert.Equal(t, "Group [MyGroup] deleted.\n", output)
	config.Client = nil
}

func TestGroupCloneMissingFieldsFails(t *testing.T) {
	cmd := NewGroupsCommand()
	_, err := test.ExecuteCommand(cmd, "clone",
		"--name=MyGroup",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestGroupCloneHttpFail(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGroupsCommand()
	_, err = test.ExecuteCommand(cmd, "clone",
		"--name=MyGroup",
		"--cloneName=MyClonedGroup",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestGroupCloneSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGroupsCommand()
	output, err := test.ExecuteCommand(cmd, "clone",
		"--name=MyGroup",
		"--cloneName=MyClonedGroup",
	)
	assert.Nil(t, err)
	assert.Equal(t, "Group [MyGroup] cloned to [MyClonedGroup].\n", output)
	config.Client = nil
}
