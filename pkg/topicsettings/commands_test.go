package topicsettings

import (
	"net/http"
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/test"
	"github.com/stretchr/testify/assert"
)

const topicsettingsresponse = `
{
	"partitions": {
		"min": 6,
		"max": 9
	},
	"replication": {
		"min": 2,
		"max": 3
	},
	"retention": {
		"size": {
		"default": 1024,
		"max": -1
		},
		"time": {
		"default": 1640000,
		"max": 24200000
		}
	}
}
`

func TestNewTopicSettingsCmdSuccess(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(topicsettingsresponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewTopicSettingsCmd()

	var outputValue string

	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.NotEmpty(t, output)

	config.Client = nil
}

func TestTopicSettingsCmdFail(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewTopicSettingsCmd()

	var outputValue string

	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	_, err = test.ExecuteCommand(cmd)

	assert.NotNil(t, err)

	config.Client = nil
}

func TestUpdateTopicSettingsCmdSuccess(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := UpdateTopicSettingsCmd()
	output, err := test.ExecuteCommand(cmd, "update",
		"--partitions-min=1",
		"--replication-min=2",
		"--retention-size-default=-1",
		"--retention-size-max=-1",
		"--retention-time-default=-1",
		"--retention-time-max=-1",
	)

	assert.Nil(t, err)
	assert.Empty(t, output)

	config.Client = nil
}

func TestUpdateTopicSettingsCmdFailWithMissingFields(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := UpdateTopicSettingsCmd()
	_, err = test.ExecuteCommand(cmd, "update",
		"--partitions-min=1",
		"--replication-min=2",
	)

	assert.NotNil(t, err)
	config.Client = nil
}

func TestUpdateTopicSettingsCmdFailMissingDescriptionButNotPattern(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := UpdateTopicSettingsCmd()
	_, err = test.ExecuteCommand(cmd, "update",
		"--partitions-min=1",
		"--replication-min=2",
		"--retention-time-max=-1",
		"--retention-size-max=-1",
		"--naming-pattern=[a-zA-Z]*",
	)

	assert.Error(t, err)
	config.Client = nil
}

func TestUpdateTopicSettingsCmdFailMissingPatternButNotDe(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := UpdateTopicSettingsCmd()
	_, err = test.ExecuteCommand(cmd, "update",
		"--partitions-min=1",
		"--replication-min=2",
		"--retention-time-max=-1",
		"--retention-size-max=-1",
		"--naming-description=[a-zA-Z]*",
	)

	assert.Error(t, err)
	config.Client = nil
}
