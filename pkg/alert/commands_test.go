package alert

import (
	"errors"
	"net/http"
	"testing"

	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/test"
	"github.com/stretchr/testify/assert"
)

func TestNewUpdateAlertSettingsCommand(t *testing.T) {
	testsAlertSettingSetCmd := []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"Missing `id` param",
			[]string{},
			"",
			errors.New("requires `id` parameter"),
		},
		{
			"Missing `channels` param",
			[]string{"--id=1000"},
			"",
			errors.New("requires `channels` parameter"),
		},
		{
			"Missing `enable` param",
			[]string{"--id=1000", "--channels='143315dd-80bf-4833-a13a-394be06dda87'"},
			"Update alert's setting has succeeded",
			errors.New(""),
		},
	}

	for _, tt := range testsAlertSettingSetCmd {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(nil))
		})
		httpClient, teardown := test.TestingHTTPClient(h)
		defer teardown()
		client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
		assert.Nil(t, err)
		config.Client = client

		t.Run(tt.name, func(t *testing.T) {
			cmd := NewUpdateAlertSettingsCommand()
			out, err := test.ExecuteCommand(cmd, tt.args...)

			test.CheckStringContains(t, out, tt.expectOut)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("%v: got `%v`, want `%v`", tt.name, err, tt.expectError)
				return
			}
			if err == nil && tt.expectError.Error() != "" {
				t.Errorf("%v: got `%v`, want `%v`", tt.name, err, tt.expectError)
				return
			}
		})
	}
}

func TestCreateOrUpdateAlertSettingConditionCommand(t *testing.T) {
	testsAlertSettingConditionSetCmd := []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"Missing `alert` param/flag",
			[]string{"--condition='69'"},
			"",
			errors.New("required flag \"alert\" not set"),
		},
		{
			"Create new condition",
			[]string{"--alert=2000", "--condition='69'"},
			"Condition [id=2000] added",
			errors.New(""),
		},
		{
			"Update a rule's channels",
			[]string{"--alert=2000", "--condition='69'", "--conditionID='6969'", "--channels='1234'"},
			"Update rule's channels succeeded",
			errors.New(""),
		},
		//"Data produced" tests
		{
			"Missing topic flag",
			[]string{"--alert=5000"},
			"",
			errors.New(`required flag "topic" not set`),
		},
		{
			`Flags "more-than" or "less-than" not set`,
			[]string{"--alert=5000", "--topic=foo"},
			"",
			errors.New(`required flag "more-than" or "less-than" not set`),
		},
		{
			`Setting both flags "more-than" and "less-than"`,
			[]string{"--alert=5000", "--topic=foo", "--more-than=5", "--less-than=6"},
			"",
			errors.New(`only one flag of "more-than" or "less-than" is supported`),
		},
		{
			`Seting wrong value for flag "more-than"`,
			[]string{"--alert=5000", "--topic=foo", "--more-than=-5", "--duration=PT2H"},
			"",
			errors.New(`"more-than" flag should be greater than zero`),
		},
		{
			`Seting wrong value for flag "less-than"`,
			[]string{"--alert=5000", "--topic=foo", "--less-than=-5", "--duration=PT2H"},
			"",
			errors.New(`"less-than" flag should be greater than zero`),
		},
		{
			`Missing "duration" flag"`,
			[]string{"--alert=5000", "--topic=foo", "--more-than=5"},
			"",
			errors.New(`required flag "duration" not set`),
		},
	}

	for _, tt := range testsAlertSettingConditionSetCmd {
		h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(nil))
		})
		httpClient, teardown := test.TestingHTTPClient(h)
		defer teardown()
		client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
		assert.Nil(t, err)
		config.Client = client

		t.Run(tt.name, func(t *testing.T) {
			cmd := NewSetAlertSettingConditionCommand()
			out, err := test.ExecuteCommand(cmd, tt.args...)

			test.CheckStringContains(t, out, tt.expectOut)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("%v: got `%v`, want `%v`", tt.name, err, tt.expectError)
				return
			}
			if err == nil && tt.expectError.Error() != "" {
				t.Errorf("%v: got `%v`, want `%v`", tt.name, err, tt.expectError)
				return
			}
		})
	}
}

func TestServerFailures(t *testing.T) {
	var testsServerFailures = []struct {
		name        string
		args        []string
		expectOut   string
		expectError error
	}{
		{
			"Server failure for updating alert's settings",
			[]string{"set", "--id=1000", "--channels='143315dd-80bf-4833-a13a-394be06dda87'"},
			"",
			errors.New("failed to update an alert's settings. Error: [response returned status code 400]"),
		},
		{
			"Server failure for updating alert's settings condition",
			[]string{"condition", "set", "--alert=2000", "--condition='69'"},
			"",
			errors.New("failed to create or update an alert's condition. Error: [response returned status code 400]"),
		},
		{
			"Server failure for updating alert's settings condition with the new flags",
			[]string{"condition", "set", "--alert=2000", "--condition='69'", "--conditionID='6969'", "--channels='1234'"},
			"",
			errors.New("failed to update alert's condition. Error: [response returned status code 400]"),
		},
	}

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
			cmd := NewAlertSettingGroupCommand()
			out, err := test.ExecuteCommand(cmd, tt.args...)
			test.CheckStringContains(t, out, tt.expectOut)
			if err != nil && err.Error() != tt.expectError.Error() {
				t.Errorf("got `%v`, want `%v`", err, tt.expectError)
			}
		})
	}
}
