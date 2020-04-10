package consumers

import (
	"errors"
	"net/http"
	"testing"

	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/test"
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
