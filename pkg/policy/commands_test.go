package policy

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	test "github.com/landoop/lenses-go/test"
	"github.com/stretchr/testify/assert"
)

const policiesOkResponse = `[
	{
	  "id": "0",
	  "name": "TestPolicy",
	  "category": "Address",
	  "impact": {
		"apps": [],
		"connectors": [],
		"processors": [],
		"topics": [
		  "test-topic"
		]
	  },
	  "impactType": "HIGH",
	  "obfuscation": "Email",
	  "fields": [
		"address"
	  ],
	  "lastUpdated": "2018-12-01 16:00:01",
	  "lastUpdatedUser": "admin"
	},
	{
	  "id": "0",
	  "name": "TestPolicy2",
	  "category": "Address",
	  "impact": {
		"apps": [],
		"connectors": [],
		"processors": [],
		"topics": [
		  "test-topic"
		]
	  },
	  "impactType": "HIGH",
	  "obfuscation": "Email",
	  "fields": [
		"address"
	  ],
	  "lastUpdated": "2018-12-01 16:00:01",
	  "lastUpdatedUser": "admin"
	}
  ]`

func TestPoliciesCommandSuccess(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(policiesOkResponse))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	//test `policies` cmd
	cmd := NewGetPoliciesCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	config.Client = nil
}

func TestPoliciesCommandHttpFail(t *testing.T) {
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
	cmd := NewGetPoliciesCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	_, err = test.ExecuteCommand(cmd)

	assert.NotNil(t, err)

	config.Client = nil
}

func TestPoliciesCommandByNameSuccess(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(policiesOkResponse))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	//test `policies` cmd
	cmd := NewGetPoliciesCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "--name=TestPolicy")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var policy api.DataPolicy
	err = json.Unmarshal([]byte(output), &policy)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, "TestPolicy", policy.Name)

	config.Client = nil
}

func TestPoliciesCommandByNameError(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(policiesOkResponse))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	//test `policies` cmd
	cmd := NewGetPoliciesCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	_, err = test.ExecuteCommand(cmd, "--name=test")

	assert.NotNil(t, err)

	config.Client = nil
}

const redactionsOkResponse = `[
	"All",
	"Email",
	"Initials",
	"First-1",
	"First-2",
	"First-3",
	"First-4",
	"Last-1",
	"Last-2",
	"Last-3",
	"Last-4",
	"Number-to-negative-one",
	"Number-to-zero",
	"Number-to-null",
	"None"
  ]`

func TestPoliciesCommandRedactionsSuccess(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(redactionsOkResponse))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGetPoliciesCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "redactions")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var redactions []api.DataObfuscationType
	err = json.Unmarshal([]byte(output), &redactions)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 15, len(redactions))

	config.Client = nil
}

func TestPoliciesCommandRedactionsHttpFail(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGetPoliciesCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	_, err = test.ExecuteCommand(cmd, "redactions")

	assert.NotNil(t, err)

	config.Client = nil
}

const impactTypesOkResponse = `["HIGH","MEDIUM","LOW"]`

func TestPoliciesCommandImpactTypesSuccess(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(impactTypesOkResponse))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGetPoliciesCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "impact-types")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var impactTypes []api.DataImpactType
	err = json.Unmarshal([]byte(output), &impactTypes)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, 3, len(impactTypes))
	config.Client = nil
}

func TestPoliciesCommandImpactTypesHttpFail(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGetPoliciesCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	_, err = test.ExecuteCommand(cmd, "impact-types")

	assert.NotNil(t, err)

	config.Client = nil
}

func TestPolicyViewCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(policiesOkResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewPolicyGroupCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "view", "--name=TestPolicy")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	var policy api.DataPolicy
	err = json.Unmarshal([]byte(output), &policy)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "TestPolicy", policy.Name)

	config.Client = nil
}

func TestPolicyCreateCommandFailMissingFields(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewPolicyGroupCommand()
	_, err = test.ExecuteCommand(cmd, "create", "--name=TestPolicy")

	assert.NotNil(t, err)

	config.Client = nil
}

func TestPolicyCreateCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewPolicyGroupCommand()
	output, err := test.ExecuteCommand(cmd, "create",
		"--name=MyTestPolicy",
		"--category=my-category",
		"--impact=HIGH",
		"--redaction=First-1",
		"--fields=f1,f2,f3",
	)
	assert.Nil(t, err)
	assert.Equal(t, "Policy [MyTestPolicy] created\n", output)
	config.Client = nil
}

func TestPolicyCreateCommandFail(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewPolicyGroupCommand()
	_, err = test.ExecuteCommand(cmd, "create",
		"--name=MyTestPolicy",
		"--category=my-category",
		"--impact=HIGH",
		"--redaction=First-1",
		"--fields=f1,f2,f3",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestPolicyUpdateCommandFailMissingFields(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewPolicyGroupCommand()
	_, err = test.ExecuteCommand(cmd, "update", "--name=TestPolicy")

	assert.NotNil(t, err)

	config.Client = nil
}

func TestPolicyUpdateCommandSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewPolicyGroupCommand()
	output, err := test.ExecuteCommand(cmd, "update",
		"--id=0",
		"--name=MyTestPolicy",
		"--category=my-category",
		"--impact=HIGH",
		"--redaction=First-1",
		"--fields=f1,f2,f3",
	)
	assert.Nil(t, err)
	assert.Equal(t, "Policy [MyTestPolicy] updated\n", output)
	config.Client = nil
}

func TestPolicyUpdateCommandFail(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewPolicyGroupCommand()
	_, err = test.ExecuteCommand(cmd, "update",
		"--id=0",
		"--name=MyTestPolicy",
		"--category=my-category",
		"--impact=HIGH",
		"--redaction=First-1",
		"--fields=f1,f2,f3",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestPolicyDeleteMissingFieldsFails(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewPolicyGroupCommand()
	_, err = test.ExecuteCommand(cmd, "delete",
		"--id=",
	)
	assert.NotNil(t, err)
	config.Client = nil
}

func TestPolicyDeleteSuccess(t *testing.T) {

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewPolicyGroupCommand()
	output, err := test.ExecuteCommand(cmd, "delete",
		"--id=0",
	)
	assert.Nil(t, err)
	assert.Equal(t, "Policy [0] deleted if it exists.\n", output)
	config.Client = nil
}
