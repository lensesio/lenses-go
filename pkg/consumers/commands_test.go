package consumers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/test"
	"github.com/stretchr/testify/assert"
)

var (
	tests = []struct {
		name        string
		args        []string
		expectError error
	}{
		{"Run `consumers` command", []string{""}, nil},
		{"Run `offsets` subcommand", []string{"offsets"}, nil},
	}

	testsDeleteConsumerGroup = []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"Deleting a consumer group",
			[]string{"delete-consumer-group", "--group", "foo-group"},
			deleteConsumerGroupCmdSuccess,
			nil,
		},
		{
			"Skipping a necessary flag",
			[]string{"delete-consumer-group"},
			"",
			errMissingConsumerGroupFlag,
		},
	}

	testsDeleteSingleTopicsOffsets = []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"Delete single offset",
			[]string{
				"offsets", "delete-single-partition-offsets", "--group", "foo-group", "--topic", "foo",
				"--partition", "1"},
			deleteSingleOffsetCmdSuccess,
			nil,
		},
		{
			"Delete single offset with multiple `topic` flags",
			[]string{
				"offsets", "delete-single-partition-offsets", "--group", "foo-group", "--topic", "foo",
				"--topic", "foo2", "--partition", "1"},
			"",
			errMultipleTopics,
		},
		{
			"Skipping offset flag",
			[]string{
				"offsets", "delete-single-partition-offsets", "--group", "foo-group", "--topic", "foo"},
			"",
			errMissingConsumerPartitionFlag,
		},
		{
			"Skipping `topic` flag",
			[]string{
				"offsets", "delete-single-partition-offsets", "--group", "foo-group", "--partition", "1"},
			errTopicMissing.Error(),
			nil,
		},
	}

	testsDeleteMultipleTopicsOffests = []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"Delete multiple topics offset",
			[]string{
				"offsets", "delete-multiple-topics-offsets", "--group", "foo-group", "--topic", "foo"},
			deleteMultipleOffsetsCmdSuccess,
			nil,
		},
		{
			"Delete multiple topics offset with all-topics flag",
			[]string{
				"offsets", "delete-multiple-topics-offsets", "--group", "foo-group", "--all-topics"},
			deleteMultipleOffsetsCmdSuccess,
			nil,
		},
		{
			"Delete multiple offset with multiple `topic` flags",
			[]string{
				"offsets", "delete-multiple-topics-offsets", "--group", "foo-group", "--topic", "foo",
				"--topic", "foo2"},
			deleteMultipleOffsetsCmdSuccess,
			nil,
		},
		{
			"Skipping `group` flag",
			[]string{
				"offsets", "delete-multiple-topics-offsets", "--topic", "foo"},
			"",
			errMissingConsumerGroupFlag,
		},
		{
			"Skipping `topic` flag",
			[]string{
				"offsets", "delete-multiple-topics-offsets", "--group", "foo-group"},
			errTopicsMissing.Error(),
			nil,
		},
	}

	testsSingleTopic = []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"Setting `to-offset` flag",
			[]string{
				"offsets", "update-single-partition", "--group", "foo-group", "--topic", "foo",
				"--partition", "1", "--to-offset", "1"},
			updateSingleCmdSuccess,
			nil,
		},
		{
			"Setting `to-earliest` flag",
			[]string{
				"offsets", "update-single-partition", "--group", "foo-group", "--topic", "foo",
				"--partition", "1", "--to-earliest"},
			updateSingleCmdSuccess,
			nil,
		},
		{
			"Setting `to-latest` flag",
			[]string{
				"offsets", "update-single-partition", "--group", "foo-group", "--topic", "foo",
				"--partition", "1", "--to-latest"},
			updateSingleCmdSuccess,
			nil,
		},
		{
			"Setting multiple `topic` flags",
			[]string{
				"offsets", "update-single-partition", "--group", "foo-group", "--topic", "foo",
				"--topic", "foo2", "--partition", "1", "--to-latest"},
			"",
			errMultipleTopics,
		},
		{
			"Skipping a necessary flag",
			[]string{
				"offsets", "update-single-partition", "--group", "foo-group", "--topic", "foo",
				"--partition", "1"},
			"",
			errMissingSinglePartitionFlag,
		},
		{
			"Skipping `topic` flag",
			[]string{
				"offsets", "update-single-partition", "--group", "foo-group", "--partition", "1"},
			errTopicMissing.Error(),
			nil,
		},
	}

	testsMultipleTopic = []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"Skipping necessary parent flags for `update-multiple-partitions` command",
			[]string{
				"offsets", "update-multiple-partitions"},
			"",
			errors.New("required flag(s) \"group\" not set"),
		},
		{
			"Setting `to-datetime` flag",
			[]string{
				"offsets", "update-multiple-partitions", "--group", "foo-group", "--topic", "foo",
				"--to-datetime", "1"},
			updateMultipleCmdSuccess,
			nil,
		},
		{
			"Setting `to-earliest` flag",
			[]string{
				"offsets", "update-multiple-partitions", "--group", "foo-group", "--topic", "foo",
				"--to-earliest"},
			updateMultipleCmdSuccess,
			nil,
		},
		{
			"Setting `to-latest` flag",
			[]string{
				"offsets", "update-multiple-partitions", "--group", "foo-group", "--topic", "foo",
				"--to-latest"},
			updateMultipleCmdSuccess,
			nil,
		},
		{
			"Skipping necessary local flags for `update-multiple-partitions` command",
			[]string{
				"offsets", "update-multiple-partitions", "--group", "foo-group", "--topic", "foo"},
			"",
			errMissingMultiplePartitionsFlag,
		},
		{
			"Skipping `topics` flag",
			[]string{
				"offsets", "update-multiple-partitions", "--group", "foo-group", "--to-latest"},
			errTopicsMissing.Error(),
			nil,
		},
		{
			"Setting `all-topics` flag",
			[]string{
				"offsets", "update-multiple-partitions", "--group", "foo-group", "--all-topics", "--to-latest"},
			updateMultipleCmdSuccess,
			nil,
		},
	}

	testsServerFailures = []struct {
		name        string
		args        []string
		httpError   int
		expectOut   string
		expectError error
	}{
		{
			"Receive a 400 for `update-single-partition` subcommand",
			[]string{
				"offsets", "update-single-partition", "--group", "foo-group", "--topic", "foo",
				"--partition", "1", "--to-offset", "1"},
			400,
			updateSingleCmdFailure,
			nil,
		},
		{
			"Receive a 400 for `update-multiple-partitions` subcommand",
			[]string{
				"offsets", "update-multiple-partitions", "--group", "foo-group", "--topic", "foo",
				"--to-datetime", "1"},
			400,
			updateMultipleCmdFailure,
			nil,
		},
	}
)

func TestConsumersOffset(t *testing.T) {
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			consumersCmd := NewRootCommand()
			_, err := test.ExecuteCommand(consumersCmd, tt.args...)
			if err != tt.expectError {
				t.Errorf("got %v, want %v", err, tt.expectError)
			}
		})
	}
}
func TestUpdateSingleTopicOffset(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nil))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
	assert.Nil(t, err)
	config.Client = client

	for _, tt := range testsSingleTopic {
		t.Run(tt.name, func(t *testing.T) {
			consumersCmd := NewRootCommand()
			out, err := test.ExecuteCommand(consumersCmd, tt.args...)
			test.CheckStringContains(t, out, tt.expectOut)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
			}
		})
	}
}

func TestDeleteConsumerGroup(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nil))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
	assert.Nil(t, err)
	config.Client = client

	for _, tt := range testsDeleteConsumerGroup {
		t.Run(tt.name, func(t *testing.T) {
			consumersCmd := NewRootCommand()
			out, err := test.ExecuteCommand(consumersCmd, tt.args...)
			test.CheckStringContains(t, out, tt.expectOut)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
			}
		})
	}
}

func TestDeleteSingleConsumerGroupOffsets(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nil))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
	assert.Nil(t, err)
	config.Client = client

	for _, tt := range testsDeleteSingleTopicsOffsets {
		t.Run(tt.name, func(t *testing.T) {
			consumersCmd := NewRootCommand()
			out, err := test.ExecuteCommand(consumersCmd, tt.args...)
			test.CheckStringContains(t, out, tt.expectOut)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
			}
		})
	}
}

func TestDeleteMultipleConsumerGroupOffsets(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nil))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
	assert.Nil(t, err)
	config.Client = client

	for _, tt := range testsDeleteMultipleTopicsOffests {
		t.Run(tt.name, func(t *testing.T) {
			consumersCmd := NewRootCommand()
			out, err := test.ExecuteCommand(consumersCmd, tt.args...)
			test.CheckStringContains(t, out, tt.expectOut)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
			}
		})
	}
}

func TestUpdateMultipleTopicsOffset(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(nil))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
	assert.Nil(t, err)
	config.Client = client

	for _, tt := range testsMultipleTopic {
		t.Run(tt.name, func(t *testing.T) {
			consumersCmd := NewRootCommand()
			out, err := test.ExecuteCommand(consumersCmd, tt.args...)
			test.CheckStringContains(t, out, tt.expectOut)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
			}
		})
	}
}

func TestServerFailures(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(nil))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()
	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
	assert.Nil(t, err)
	config.Client = client

	for _, tt := range testsServerFailures {
		t.Run(tt.name, func(t *testing.T) {
			consumersCmd := NewRootCommand()
			out, _ := test.ExecuteCommand(consumersCmd, tt.args...)
			test.CheckStringContains(t, out, tt.expectOut)
		})
	}
}

func TestContextCommands(t *testing.T) {
	scenarios := make(map[string]test.CommandTest)

	scenarios["Run `offsets update-single-partition` subcommand without params should throw error"] =
		test.CommandTest{
			Cmd:     NewRootCommand,
			CmdArgs: []string{"offsets", "update-single-partition"},
			ShouldContainErrors: []string{
				`required flag(s) "group", "partition" not set`,
			},
			ShouldContain: []string{
				`required flag(s) "group", "partition" not set`,
			},
		}

	test.RunCommandTests(t, scenarios)
}
