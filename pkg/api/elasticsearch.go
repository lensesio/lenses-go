package api

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/lensesio/lenses-go/pkg"
)

// Shard type for elasticsearch shards
type Shard struct {
	Shard             string `json:"shard"`
	Records           int    `json:"records"`
	Replicas          int    `json:"replicas"`
	AvailableReplicas int    `json:"availableReplicas"`
}

// Index is Elasticsearch index type
type Index struct {
	IndexName      string   `json:"indexName" header:"Name"`
	ConnectionName string   `json:"connectionName" header:"Connection"`
	KeyType        string   `json:"keyType"`
	ValueType      string   `json:"valueType"`
	KeySchema      string   `json:"keySchema,omitempty"`
	ValueSchema    string   `json:"valueSchema,omitempty"`
	Size           int      `json:"size" header:"Size"`
	TotalRecords   int      `json:"totalMessages" header:"Records"`
	Description    string   `json:"description" yaml:"description"`
	Status         string   `json:"status" header:"Status"`
	Shards         []Shard  `json:"shards"`
	ShardsCount    int      `json:"shardsCount" header:"Shards"`
	Replicas       int      `json:"replicas" header:"Replicas"`
	Permission     []string `json:"permissions"`
}

// GetIndexes returns the list of elasticsearch indexes.
func (c *Client) GetIndexes(connectionName string, includeSystemIndexes bool) (indexes []Index, err error) {
	// # List of indexes
	// GET /api/elastic/indexes?connectionName=$x&includeSystemIndexes=$y
	url, err := url.Parse(pkg.ElasticsearchIndexesPath)
	q := url.Query()

	q.Add("includeSystemIndexes", strconv.FormatBool(includeSystemIndexes))

	if connectionName != "" {
		q.Add("connectionName", connectionName)
	}
	url.RawQuery = q.Encode()

	resp, respErr := c.Do(http.MethodGet, url.String(), "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &indexes)

	return
}

// GetIndex fetches stuff about an index
func (c *Client) GetIndex(connectionName string, indexName string) (index Index, err error) {
	// List of indexes
	// GET /api/elastic/indexes/connectionName/indexName
	path := fmt.Sprintf("%s/%s/%s", pkg.ElasticsearchIndexesPath, connectionName, indexName)

	resp, respErr := c.Do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &index)

	return
}

// GetAvailableReplicas returns the sum of all shards' available replicas
func GetAvailableReplicas(esIndex Index) int {
	availableReplicas := 0

	for _, shard := range esIndex.Shards {
		availableReplicas = availableReplicas + shard.AvailableReplicas
	}

	return availableReplicas
}
