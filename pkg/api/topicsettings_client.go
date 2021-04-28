package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

// MinMax contains Min and Max keys
type MinMax struct {
	Min int `json:"min" yaml:"min"`
	Max int `json:"max,omitempty" yaml:"max,omitempty"`
}

//DefaultMax contains Default and Max keys
type DefaultMax struct {
	Default int64 `json:"default,omitempty" yaml:"default,omitempty"`
	Max     int64 `json:"max" yaml:"max"`
}

// Retention contains Size and Time keys
type Retention struct {
	Size DefaultMax `json:"size" yaml:"size"`
	Time DefaultMax `json:"time" yaml:"time"`
}

// TopicConfiguration contains Partitions, Replication and Retention keys
type TopicConfiguration struct {
	Partitions  MinMax    `json:"partitions" yaml:"partitions"`
	Replication MinMax    `json:"replication" yaml:"replication"`
	Retention   Retention `json:"retention" yaml:"retention"`
}

// Naming contains Description and Pattern
type Naming struct {
	Description string `json:"description" yaml:"description"`
	Pattern     string `json:"pattern" yaml:"pattern"`
}

// TopicSettingsResponse contains Config, Naming and IsApplicable keys
type TopicSettingsResponse struct {
	Config       TopicConfiguration `json:"config" yaml:"config"`
	Naming       *Naming            `json:"naming,omitempty" yaml:"naming,omitempty"`
	IsApplicable bool               `json:"isApplicable,omitempty" yaml:"isApplicable,omitempty"`
}

// TopicSettingsRequest contains Config and Naming keys
type TopicSettingsRequest struct {
	Config TopicConfiguration `json:"config" yaml:"config"`
	Naming *Naming            `json:"naming" yaml:"naming"`
}

const path = "api/v1/kafka/topic/policy"

// GetTopicSettings from the API
func (c *Client) GetTopicSettings() (settings TopicSettingsResponse, err error) {
	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}

	err = c.ReadJSON(resp, &settings)
	return
}

// UpdateTopicSettings from the API
func (c *Client) UpdateTopicSettings(settings TopicSettingsRequest) error {
	if settings.Config.Partitions.Min < 1 {
		return fmt.Errorf("Partitions cannot have negative value")
	}

	if settings.Config.Replication.Min < 1 {
		return fmt.Errorf("Replication cannot have negative value")
	}

	if settings.Config.Retention.Size.Default < -1 {
		return fmt.Errorf("Retention size cannot have value lower than -1")
	}

	if settings.Config.Retention.Size.Max < -1 {
		return fmt.Errorf("Retention size cannot have value lower than -1")
	}

	if settings.Config.Retention.Time.Default < -1 {
		return fmt.Errorf("Retention time cannot have value lower than -1")
	}

	if settings.Config.Retention.Time.Max < -1 {
		return fmt.Errorf("Retention time cannot have value lower than -1")
	}

	payload, err := json.Marshal(settings)

	if err != nil {
		return errors.Wrap(err, "Failed to read settings from input")
	}

	_, err = c.Do(http.MethodPut, path, contentTypeJSON, payload)
	return err
}
