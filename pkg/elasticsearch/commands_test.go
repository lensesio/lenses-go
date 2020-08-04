package elasticsearch

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/test"
	"github.com/stretchr/testify/assert"
)

func index(name string, connection string) api.Index {
	var shards []api.Shard
	var permissions []string
	return api.Index{
		IndexName:      name,
		ConnectionName: connection,
		KeyType:        "STRING",
		ValueType:      "JSON",
		KeySchema:      "string",
		ValueSchema:    "null",
		Size:           4244452,
		TotalRecords:   192910,
		Status:         "yellow",
		Shards:         shards,
		ShardsCount:    0,
		Replicas:       1,
		Permission:     permissions,
	}
}

func esIndexHandler(t *testing.T, indexes []api.Index) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var filteredIndexes []api.Index
		q := r.URL.Query()

		includeSystemIndexes, err := strconv.ParseBool(q.Get("includeSystemIndexes"))
		connectionName := q.Get("connectionName")
		assert.Nil(t, err)

		for _, index := range indexes {
			if strings.HasPrefix(index.IndexName, ".") && includeSystemIndexes == false {
				continue
			}
			if connectionName != "" && index.ConnectionName != connectionName {
				continue
			}

			filteredIndexes = append(filteredIndexes, index)
		}

		resp, err := json.Marshal(filteredIndexes)
		assert.Nil(t, err)
		w.Write([]byte(resp))
	}
}

func TestIndexesCommandSuccess(t *testing.T) {
	var indexes, actual, expected []api.Index

	internal := index(".internal", "es1")
	indexA := index("indexA", "es1")
	indexB := index("indexB", "es2")

	indexes = append(indexes, internal, indexA, indexB)

	h := http.HandlerFunc(esIndexHandler(t, indexes))

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	var outputValue string
	cmd := IndexesCommand()

	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	output, err := test.ExecuteCommand(cmd)
	assert.Nil(t, err)

	expected = append(expected, indexA, indexB)
	err = json.Unmarshal([]byte(output), &actual)
	assert.Nil(t, err)
	assert.EqualValues(t, &expected, &actual)
	config.Client = nil
}

func TestIndexesCommandWithParamsSuccess(t *testing.T) {
	var indexes, actual, expected []api.Index

	internal := index(".internal", "es1")
	indexA := index("indexA", "es1")
	indexB := index("indexB", "es2")

	indexes = append(indexes, internal, indexA, indexB)

	h := http.HandlerFunc(esIndexHandler(t, indexes))

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	var outputValue string
	cmd := IndexesCommand()

	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	output, err := test.ExecuteCommand(cmd, "--connection=es1", "--include-system-indexes")
	assert.Nil(t, err)

	expected = append(expected, internal, indexA)
	err = json.Unmarshal([]byte(output), &actual)
	assert.Nil(t, err)
	assert.EqualValues(t, &expected, &actual)
	config.Client = nil
}

func TestIndexesCommandFail(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	_, indexErr := test.ExecuteCommand(IndexesCommand())

	assert.NotNil(t, indexErr)
	config.Client = nil
}

func TestIndexCommand(t *testing.T) {
	indexA := index("indexA", "es1")

	resp, err := json.Marshal(indexA)

	assert.Nil(t, err)

	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(resp))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := IndexCommand()

	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	output, err := test.ExecuteCommand(cmd, `--connection="lorem"`, `--name="ipsum"`)

	assert.NotEmpty(t, output)
	assert.Nil(t, err)

	config.Client = nil
}

func TestIndexCommendFail(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := IndexCommand()

	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	_, indexErr := test.ExecuteCommand(cmd, `--connection="lorem"`, `--name="ipsum"`)

	assert.NotNil(t, indexErr)
	config.Client = nil
}
