package api

import (
	"encoding/json"
	"testing"
)

func TestAPIConfig(t *testing.T) {
	apiConfigTests := []struct {
		name         string
		httpResponse string
		expectValue  int
	}{
		{
			"Empty string must unmarshal to 0 int value",
			`{"lenses.jmx.port":""}`,
			0,
		},
		{
			"Any integer should deserialize to int",
			`{"lenses.jmx.port":6}`,
			6,
		},
		{
			"Any int passed as string should deserialize to int",
			`{"lenses.jmx.port":"69"}`,
			69,
		},
		{
			"null should deserialize to 0 int value",
			`{"lenses.jmx.port":null}`,
			0,
		},
	}

	for _, tt := range apiConfigTests {
		var cfg BoxConfig

		err := json.Unmarshal([]byte(tt.httpResponse), &cfg)
		if err != nil {
			t.Error("failed to unmarshal: ", err)
			return
		}

		if tt.expectValue != int(cfg.JMXPort) {
			t.Error(tt.name)
			t.Errorf("got `%v`, want `%v`", cfg.JMXPort, tt.expectValue)
		}
	}
}
