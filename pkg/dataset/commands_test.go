package dataset

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/test"
	"github.com/stretchr/testify/assert"
)

const datasetResponse = `
{
  "description": null,
  "topicName": "nyc_yellow_taxi_trip_data",
}
`

func TestNewDatasetGroupCmdSuccess(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(datasetResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := NewDatasetGroupCmd()
	var outputValue string

	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.NoError(t, err)
	assert.NotEmpty(t, output)
}

func TestNewDatasetUpdateDescriptionCmdSuccess(t *testing.T) {
	var payload api.UpdateDatasetDescription
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		json.Unmarshal(body, &payload)
		w.Write([]byte(""))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := UpdateDatasetDescriptionCmd()
	_, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--name=topicName",
		"--description=Some Description",
	)

	assert.NoError(t, err)
	assert.Equal(t, "Some Description", payload.Description)

	config.Client = nil
}

func TestNewDatasetUpdateRejectsBlankDescription(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(""))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := UpdateDatasetDescriptionCmd()
	_, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--name=topicName",
		"--description= \n \t  ",
	)

	assert.Error(t, err)
	config.Client = nil
}

func TestDatasetRemoveDescription(t *testing.T) {
	var payload map[string]string
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		json.Unmarshal(body, &payload)
		w.Write([]byte(""))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := RemoveDatasetDescriptionCmd()
	_, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--name=topicName",
	)

	assert.NoError(t, err)
	assert.Equal(t, map[string]string{}, payload)
	config.Client = nil
}

func TestNewDatasetUpdateMetadataCmdFailureNoConnection(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(datasetResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := UpdateDatasetDescriptionCmd()
	output, err := test.ExecuteCommand(cmd,
		"--name=topicName",
		"--description=Some Description",
	)

	assert.Error(t, err)
	assert.NotEmpty(t, output)

	config.Client = nil
}

func TestNewDatasetUpdateMetadataCmdFailureNoName(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(datasetResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := UpdateDatasetDescriptionCmd()
	output, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--description=Some Description",
	)

	assert.Error(t, err)
	assert.NotEmpty(t, output)

	config.Client = nil
}
