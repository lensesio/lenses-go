package elasticsearch

import (
	"testing"

	"github.com/landoop/lenses-go/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestMakeIndexView(t *testing.T) {
	shard := api.Shard{Shard: "1", Records: 2, Replicas: 1, AvailableReplicas: 1}

	apiResponse := api.Index{
		Shards: []api.Shard{shard, shard},
	}

	indexView := MakeIndexView(apiResponse)

	assert.Equal(t, 2, indexView.ShardsCount)
}

func TestMakeIndexViewNoShards(t *testing.T) {
	apiResponse := api.Index{}
	indexView := MakeIndexView(apiResponse)

	assert.Equal(t, 0, indexView.ShardsCount)
}
