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

const usersOkResponse = `
[
  {
    "username": "sam",
    "email": "sam@landoop.com",
    "groups": [
      "foo",
      "bar"
    ]
  },
  {
    "username": "stef",
    "groups": [
      "boo"
    ],
    "type": "BASIC"
  }
]`

func TestUsersCommandSuccess(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(usersOkResponse))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	//test `users` cmd
	cmd := NewUsersCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	config.Client = nil
}

func TestFilteredUsersCommandSuccess(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(usersOkResponse))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	//test `users` cmd
	cmd := NewUsersCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "--groups=boo")

	assert.Nil(t, err)
	var users []api.UserMember
	err = json.Unmarshal([]byte(output), &users)

	assert.Nil(t, err)
	assert.Equal(t, 1, len(users))
	assert.Equal(t, "stef", users[0].Username)

	config.Client = nil
}

func TestUsersCommandHttpFail(t *testing.T) {
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
	cmd := NewUsersCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	_, err = test.ExecuteCommand(cmd)

	assert.NotNil(t, err)

	config.Client = nil
}

const userOkResponse = `{
	"username": "sam",
	"email": "sam@landoop.com",
	"groups": [
	  "foo",
	  "bar"
	],
	"type": "BASIC"
  }`

func TestUserViewCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(userOkResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewUsersCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "get", "--username=sam")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var user api.UserMember
	err = json.Unmarshal([]byte(output), &user)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "sam", user.Username)
	assert.Equal(t, "sam@landoop.com", user.Email)
	assert.Equal(t, "BASIC", user.Type)
	assert.Equal(t, 2, len(user.Groups))

	config.Client = nil
}

func TestUsersCreateCommandFailMissingFields(t *testing.T) {
	cmd := NewUsersCommand()
	_, err := test.ExecuteCommand(cmd, "create", "--username=MyGroup")

	assert.NotNil(t, err)

	config.Client = nil
}

func TestUsersCreateCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewUsersCommand()
	output, err := test.ExecuteCommand(cmd, "create",
		"--username=spiros",
		"--password=secret",
		"--security=basic",
		"--groups=MyGroup",
	)
	assert.Nil(t, err)
	assert.Equal(t, "User [spiros] created\n", output)
	config.Client = nil
}

func TestUsersUpdateCommandFailMissingFields(t *testing.T) {

	cmd := NewUsersCommand()
	_, err := test.ExecuteCommand(cmd, "update", "--username=spiros")

	assert.NotNil(t, err)

	config.Client = nil
}

func TestUsersUpdateCommandHttpFail(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewUsersCommand()
	_, err = test.ExecuteCommand(cmd, "update",
		"--username=spiros",
		"--groups=MyGroup",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestUsersUpdateCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewUsersCommand()
	output, err := test.ExecuteCommand(cmd, "update",
		"--username=spiros",
		"--groups=MyGroup",
	)
	assert.Nil(t, err)
	assert.Equal(t, "User [spiros] updated\n", output)
	config.Client = nil
}

func TestUsersDeleteMissingFieldsFails(t *testing.T) {
	cmd := NewUsersCommand()
	_, err := test.ExecuteCommand(cmd, "delete",
		"--username=",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestUsersDeleteHttpFails(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewUsersCommand()
	_, err = test.ExecuteCommand(cmd, "delete",
		"--username=spiros",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestUsersDeleteSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewUsersCommand()
	output, err := test.ExecuteCommand(cmd, "delete",
		"--username=spiros",
	)
	assert.Nil(t, err)
	assert.Equal(t, "User [spiros] deleted.\n", output)
	config.Client = nil
}

func TestUsersUpdatePasswordFieldsFails(t *testing.T) {
	cmd := NewUsersCommand()
	_, err := test.ExecuteCommand(cmd, "password",
		"--username=",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestUsersUpdatePasswordHttpFails(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewUsersCommand()
	_, err = test.ExecuteCommand(cmd, "password",
		"--username=spiros",
		"--secret=secret",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestUsersUpdatePaswordSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewUsersCommand()
	output, err := test.ExecuteCommand(cmd, "password",
		"--username=spiros",
		"--secret=secret",
	)
	assert.Nil(t, err)
	assert.Equal(t, "User password [spiros] updated.\n", output)
	config.Client = nil
}
