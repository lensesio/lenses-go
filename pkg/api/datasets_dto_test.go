package api

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// JSON as obtained from the Lenses v5 API.
const listDatasetsJSON = `{
	"datasets": {
	  "values": [
		{
		  "name": "shipsaggs",
		  "highlights": [
			{
			  "fieldName": "name",
			  "startIndex": 0,
			  "endIndex": 4,
			  "arrayIndex": 0
			}
		  ],
		  "records": 347666906,
		  "recordsPerSecond": 0,
		  "keyType": "TWLONG",
		  "valueType": "AVRO",
		  "connectionName": "kafka",
		  "replication": 1,
		  "consumers": 0,
		  "partitions": 10,
		  "fields": {
			"key": [],
			"value": []
		  },
		  "isSystemEntity": false,
		  "isMarkedForDeletion": false,
		  "isCompacted": true,
		  "sizeBytes": 7667899465,
		  "policies": [],
		  "permissions": [
			"ShowTopic",
			"CreateTopic",
			"RequestTopicCreation",
			"DropTopic",
			"ConfigureTopic",
			"QueryTopic",
			"InsertData",
			"DeleteData",
			"UpdateSchema",
			"ViewSchema",
			"UpdateMetadata"
		  ],
		  "description": "test description",
		  "tags": [
			{
			  "name": "iot"
			},
			{
			  "name": "Testing"
			}
		  ],
		  "retentionMs": 604800000,
		  "retentionBytes": -1,
		  "sourceType": "Kafka"
		},
		{
		  "name": "fastships_index",
		  "highlights": [
			{
			  "fieldName": "name",
			  "startIndex": 4,
			  "endIndex": 8,
			  "arrayIndex": 0
			}
		  ],
		  "sizeBytes": 12078914278,
		  "records": 90099514,
		  "connectionName": "ESOne",
		  "replicas": 0,
		  "shard": 5,
		  "fields": {
			"key": [],
			"value": []
		  },
		  "isSystemEntity": false,
		  "policies": [
			{
			  "policyId": "542c7269-c184-4c39-83a0-912428936957",
			  "policyName": "mask MMSI",
			  "policyCategory": "PII",
			  "obfuscation": "First-3",
			  "matchingKeyFields": [],
			  "matchingValueFields": [
				{
				  "name": "MMSI",
				  "parents": []
				}
			  ]
			}
		  ],
		  "permissions": [
			"ShowIndex",
			"QueryIndex",
			"ViewSchema",
			"UpdateMetadata"
		  ],
		  "description": null,
		  "tags": [],
		  "sourceType": "Elastic"
		}
	  ],
	  "pagesAmount": 1,
	  "totalCount": 2
	},
	"sourceTypes": [
	  "Kafka",
	  "Elastic"
	]
  }
  `

// TestListDatasetsUnmarshalling especially focuses on the custom unmarshaller
// in PageDatasetMatch.
func TestListDatasetsUnmarshalling(t *testing.T) {
	var r Results
	err := json.Unmarshal([]byte(listDatasetsJSON), &r)
	require.NoError(t, err)

	// Type correctness.
	require.Len(t, r.Datasets.Values, 2)
	k, ok := r.Datasets.Values[0].(Kafka)
	require.True(t, ok)
	e, ok := r.Datasets.Values[1].(Elastic)
	require.True(t, ok)

	// Per-type unique properties.
	assert.Equal(t, 5, e.Shard)
	assert.Equal(t, "TWLONG", k.KeyType)
}
