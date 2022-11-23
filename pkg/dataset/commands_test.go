package dataset

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const datasetResponse = `
{
  "description": null,
  "topicName": "nyc_yellow_taxi_trip_data",
}
`

// stubsAPIEndpoint returns an handler for a given request method/PATH, failing back to a 501 error when these do not match.
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

	cmd := NewDatasetGroupCmd(nil)
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

type mockDatasetsClient struct {
	inListParams api.ListDatasetsParameters
	inMaxResults int
	outVs        []api.DatasetMatch
	outErr       error
}

func (m *mockDatasetsClient) ListDatasetsPg(params api.ListDatasetsParameters, maxResults int) (vs []api.DatasetMatch, err error) {
	m.inListParams = params
	m.inMaxResults = maxResults
	return m.outVs, m.outErr
}

func TestListDatasetsCmd(t *testing.T) {
	for _, stim := range []struct {
		givenArgs       []string
		expectParams    api.ListDatasetsParameters
		expectMax       int
		givenMatches    []api.DatasetMatch
		givenErr        error
		optExpectStdOut string
	}{
		{
			givenArgs: []string{"--output=plain"},
		},
		{
			givenArgs:    []string{"--output=plain", "--query=qqq"},
			expectParams: api.ListDatasetsParameters{Query: ptrTo("qqq")},
		},
		{
			givenArgs: []string{"--output=plain", "--max=42"},
			expectMax: 42,
		},
		{
			givenArgs:    []string{"--output=plain", "--has-records=true"},
			expectParams: api.ListDatasetsParameters{HasRecords: ptrTo(true)},
		},
		{
			givenArgs:    []string{"--output=plain", "--has-records=false"},
			expectParams: api.ListDatasetsParameters{HasRecords: ptrTo(false)},
		},
		{
			givenArgs:    []string{"--output=plain", "--has-records=any"},
			expectParams: api.ListDatasetsParameters{HasRecords: nil},
		},
		{
			givenArgs:    []string{"--output=plain", "--compacted=true"},
			expectParams: api.ListDatasetsParameters{Compacted: ptrTo(true)},
		},
		{
			givenArgs:    []string{"--output=plain", "--compacted=false"},
			expectParams: api.ListDatasetsParameters{Compacted: ptrTo(false)},
		},
		{
			givenArgs:    []string{"--output=plain", "--compacted=any"},
			expectParams: api.ListDatasetsParameters{Compacted: nil},
		},
		{
			givenArgs:    []string{"--output=plain", "--connections=a", "--connections=b,c"},
			expectParams: api.ListDatasetsParameters{Connections: []string{"a", "b", "c"}},
		},
		{
			givenArgs: []string{"--output=plain"},
			givenMatches: []api.DatasetMatch{
				api.Elastic{Name: "el"},
				api.Kafka{Name: "kaf"},
				api.Postgres{Name: "pg"},
				api.SchemaRegistrySubject{Name: "sr"},
			},
			optExpectStdOut: "el\nkaf\npg\nsr\n",
		},
	} {
		t.Run(strings.Join(stim.givenArgs, " "), func(t *testing.T) {
			m := &mockDatasetsClient{outVs: stim.givenMatches, outErr: stim.givenErr}
			cmd := ListDatasetsCmd(m)
			var outputValue string
			bite.RegisterOutPutFlag(cmd, &outputValue)
			stdOut, err := test.ExecuteCommand(cmd, stim.givenArgs...)
			require.NoError(t, err)
			t.Log(stdOut)
			assert.Equal(t, stim.expectParams, m.inListParams)
			assert.Equal(t, stim.expectMax, m.inMaxResults)
			if stim.optExpectStdOut != "" {
				assert.Equal(t, stim.optExpectStdOut, stdOut)
			}
		})
	}
}
