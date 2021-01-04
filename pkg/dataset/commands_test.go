package dataset

import (
	"encoding/json"
	"fmt"
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

//stubsAPIEndpoint returns an handler for a given request method/PATH, failing back to a 501 error when these do not match.
func stubAPIEndpoint(method, path string, t *testing.T, f http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == method && r.URL.Path == path {
			f(w, r)
		} else {
			errorMsg := fmt.Sprintf("Stubbed API endpoint is %s %s. Got instead %s %s", method, path, r.Method, r.URL.Path)
			t.Logf("erroring: %s", errorMsg)
			http.Error(w, errorMsg, 501)
		}
	})
}

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
	h := stubAPIEndpoint("PUT", "/api/v1/datasets/kafka/topicName/description", t, func(w http.ResponseWriter, r *http.Request) {
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

func TestDatasetUpdateTagsCmdSuccess(t *testing.T) {
	var payload api.UpdateDatasetTags
	h := stubAPIEndpoint("PUT", "/api/v1/datasets/kafka/topicName/tags", t, func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		json.Unmarshal(body, &payload)
		w.Write([]byte(""))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := UpdateDatasetTagsCmd()
	_, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--name=topicName",
		"--tag=tag1",
		"--tag=tag2",
		"--tag=tag3",
	)
	tags := []string{}
	for _, tag := range payload.Tags {
		tags = append(tags, tag.Name)
	}

	assert.NoError(t, err)
	assert.Equal(t, []string{"tag1", "tag2", "tag3"}, tags)

	config.Client = nil
}

func TestNewDatasetUpdateRejectsBlankDescription(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unexpected API call", 500)
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

func TestNewDatasetRejectBlankTags(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "Unexpected API call", 500)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := UpdateDatasetTagsCmd()
	_, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--name=topicName",
		"--tag=\t",
	)

	assert.Error(t, err)
	config.Client = nil
}

func TestDatasetRemoveDescription(t *testing.T) {
	var payload map[string]string

	h := stubAPIEndpoint("PUT", "/api/v1/datasets/kafka/topicName/description", t, func(w http.ResponseWriter, r *http.Request) {

		t.Log("Running handler")
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

func TestDatasetRemoveTags(t *testing.T) {
	var payload api.UpdateDatasetTags
	h := stubAPIEndpoint("PUT", "/api/v1/datasets/kafka/topicName/tags", t, func(w http.ResponseWriter, r *http.Request) {
		body, err := ioutil.ReadAll(r.Body)
		assert.NoError(t, err)

		json.Unmarshal(body, &payload)
		w.Write([]byte(""))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := RemoveDatasetTagsCmd()
	_, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--name=topicName",
	)

	assert.NoError(t, err)
	assert.Equal(t, []api.DatasetTag{}, payload.Tags)
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
