package dataset

import (
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

func TestNewDatasetUpdateMetadataCmdSuccess(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(datasetResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := NewDatasetUpdateMetadataCmd()
	output, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--name=topicName",
		"--description=Some Description",
	)

	assert.NoError(t, err)
	assert.Equal(t, "Lenses Metadata have been updated successfully\n", output)

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

	cmd := NewDatasetUpdateMetadataCmd()
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

	cmd := NewDatasetUpdateMetadataCmd()
	output, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--description=Some Description",
	)

	assert.Error(t, err)
	assert.NotEmpty(t, output)

	config.Client = nil
}
