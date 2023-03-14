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

const serviceAccountsOkResp = `
[
  {
    "name": "pam",
    "owner": "paul",
    "groups": [
      "foo"
    ]
  },
  {
    "name": "sam",
    "owner": null,
    "groups": [
      "bar"
    ]
  },
  {
    "name": "tim",
    "owner": null,
    "groups": [
      "foo"
    ]
  }
]`

func TestServiceAccountsCommandSuccess(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(serviceAccountsOkResp))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	//test `users` cmd
	cmd := NewServiceAccountsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	var serviceAccounts []api.ServiceAccount
	err = json.Unmarshal([]byte(output), &serviceAccounts)

	assert.Nil(t, err)
	assert.Equal(t, 3, len(serviceAccounts))

	config.Client = nil
}

func TestServiceAccountsCommandHttpFail(t *testing.T) {
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
	cmd := NewServiceAccountsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	_, err = test.ExecuteCommand(cmd)

	assert.NotNil(t, err)

	config.Client = nil
}

const serviceAccountOkResp = `{
	"name": "pam",
	"owner": "paul",
	"groups": [
	  "foo"
	]
  }`

func TestServiceAccountViewCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(serviceAccountOkResp))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewServiceAccountsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "get", "--name=pam")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var svcacc api.ServiceAccount
	err = json.Unmarshal([]byte(output), &svcacc)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "paul", svcacc.Owner)
	assert.Equal(t, 1, len(svcacc.Groups))

	config.Client = nil
}

func TestServiceAccountViewMissingFields(t *testing.T) {
	cmd := NewServiceAccountsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	_, err := test.ExecuteCommand(cmd, "get", "--name=")

	assert.NotNil(t, err)

	config.Client = nil
}

func TestServiceAccountsCreateCommandFailMissingFields(t *testing.T) {
	cmd := NewServiceAccountsCommand()
	_, err := test.ExecuteCommand(cmd, "create", "--name=svcacc")

	assert.NotNil(t, err)

	config.Client = nil
}

const serviceAccountsCreateOkResp = `
{
	"token": "4cbddcfd-a5ca-4d6e-acc5-4f5db3c9548f"
}`

func TestServiceAccountsCreateCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(serviceAccountsCreateOkResp))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewServiceAccountsCommand()
	output, err := test.ExecuteCommand(cmd, "create",
		"--name=svcacc",
		"--owner=spiros",
		"--groups=MyGroup1",
		"--groups=MyGroup2",
	)
	assert.Nil(t, err)
	assert.Equal(t, "Service Account [svcacc] created with token [4cbddcfd-a5ca-4d6e-acc5-4f5db3c9548f]\n", output)
	config.Client = nil
}

func TestServiceAccountUpdateCommandFailMissingFields(t *testing.T) {

	cmd := NewServiceAccountsCommand()
	_, err := test.ExecuteCommand(cmd, "update", "--name=spiros")

	assert.NotNil(t, err)

	config.Client = nil
}

func TestServiceAccountUpdateCommandHttpFail(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewServiceAccountsCommand()
	_, err = test.ExecuteCommand(cmd, "update",
		"--name=svcacc",
		"--owner=spiros",
		"--groups=MyGroup1",
		"--groups=MyGroup2",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestServiceAccountUpdateCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewServiceAccountsCommand()
	output, err := test.ExecuteCommand(cmd, "update",
		"--name=svcacc",
		"--owner=spiros",
		"--groups=MyGroup1",
		"--groups=MyGroup2",
	)
	assert.Nil(t, err)
	assert.Equal(t, "Service account [svcacc] updated\n", output)
	config.Client = nil
}

func TestServiceAccountDeleteMissingFieldsFails(t *testing.T) {
	cmd := NewServiceAccountsCommand()
	_, err := test.ExecuteCommand(cmd, "delete",
		"--name=",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestServiceAccountDeleteHttpFails(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewServiceAccountsCommand()
	_, err = test.ExecuteCommand(cmd, "delete",
		"--name=svcacc",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestServiceAccountDeleteSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewServiceAccountsCommand()
	output, err := test.ExecuteCommand(cmd, "delete",
		"--name=svcacc",
	)
	assert.Nil(t, err)
	assert.Equal(t, "Service account [svcacc] deleted.\n", output)
	config.Client = nil
}

func TestServiceAccountRevokeMissingFieldsFails(t *testing.T) {
	cmd := NewServiceAccountsCommand()
	_, err := test.ExecuteCommand(cmd, "revoke",
		"--name=",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestServiceAccountRevokeHttpFails(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewServiceAccountsCommand()
	_, err = test.ExecuteCommand(cmd, "revoke",
		"--name=svcacc",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestServiceAccountRevokeSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(serviceAccountsCreateOkResp))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewServiceAccountsCommand()
	output, err := test.ExecuteCommand(cmd, "revoke",
		"--name=svcacc",
	)
	assert.Nil(t, err)
	assert.Equal(t, "Service account token [svcacc] revoked. New token [4cbddcfd-a5ca-4d6e-acc5-4f5db3c9548f]\n", output)
	config.Client = nil
}
