package api

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetAvailableReplicas(t *testing.T) {
	shard := Shard{Shard: "1", Records: 2, Replicas: 1, AvailableReplicas: 1}

	index := Index{
		Shards: []Shard{shard, shard},
	}

	assert.Equal(t, 2, GetAvailableReplicas(index))
}

func TestGetAvailableReplicasEmpty(t *testing.T) {
	index := Index{
		Shards: []Shard{},
	}

	assert.Equal(t, 0, GetAvailableReplicas(index))
}
