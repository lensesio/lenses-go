package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/lensesio/lenses-go/v5/pkg"
)

// AlertSettingsPayload contains the alert's settings datastructure
type AlertSettingsPayload struct {
	AlertID  string   `json:"id,omitempty" yaml:"id"`
	Enable   bool     `json:"enable" yaml:"enable"`
	Channels []string `json:"channels" yaml:"channels"`
}

// AlertSettingsConditionPayload is the payload for creating alert conditions
type AlertSettingsConditionPayload struct {
	AlertID     string   `json:"alert,omitempty" yaml:"alert"`
	ConditionID string   `json:"conditionID,omitempty" yaml:"conditionID"`
	Condition   string   `json:"condition" yaml:"condition"`
	Channels    []string `json:"channels" yaml:"channels"`
}

// AlertConditionRequestv1 represents the schema of /api/v1/alert/settings/{alert_setting_id}/conditions payload
type AlertConditionRequestv1 struct {
	Condition DataProduced `json:"condition" yaml:"condition"`
	Channels  []string     `json:"channels" yaml:"channels"`
}

// ConsumerAlertConditionRequestv1 represents the schema of /api/v1/alert/settings/{alert_setting_id}/conditions payload for consumer
type ConsumerAlertConditionRequestv1 struct {
	Condition ConsumerConditionDsl `json:"condition" yaml:"condition"`
	Channels  []string             `json:"channels" yaml:"channels"`
}

// ConsumerConditionMode represents the consumer lag alert rule mode
type ConsumerConditionMode string

// Consumer lag alert rule modes
const (
	PerPartitionMode ConsumerConditionMode = "PerPartitionMode"
	PerTopicMode     ConsumerConditionMode = "PerTopicMode"
)

// ConsumerConditionDsl represents the consumer specific payload expected at /api/v1/alert/settings/{alert_setting_id}/conditions
type ConsumerConditionDsl struct {
	Group     string                `json:"group"`
	Threshold int                   `json:"threshold"`
	Topic     string                `json:"topic"`
	Mode      ConsumerConditionMode `json:"mode"`
}

// DataProduced is the payload for Producer's alert type category
type DataProduced struct {
	ConnectionName string    `json:"connectionName" yaml:"connectionName"`
	DatasetName    string    `json:"datasetName" yaml:"datasetName"`
	Threshold      Threshold `json:"threshold" yaml:"threshold"`
	Duration       string    `json:"duration" yaml:"duration"`
}

// Threshold corresponds to AlertSettingCondition DataProduced Threshold data structure
type Threshold struct {
	Type     string `json:"type" yaml:"type"`
	Messages int    `json:"messages" yaml:"messages"`
}

// ProducerAlertSettings is the struct used for importing/exporting alert settings
type ProducerAlertSettings struct {
	ID               int                       `json:"alert" yaml:"alert"`
	Description      string                    `json:"description" yaml:"description"`
	ConditionDetails []AlertConditionRequestv1 `json:"conditions" yaml:"conditions"`
}

// ConsumerAlertSettings is the struct used for importing/exporting consumer alert settings
type ConsumerAlertSettings struct {
	ID               int                               `json:"alert" yaml:"alert"`
	Description      string                            `json:"description" yaml:"description"`
	ConditionDetails []ConsumerAlertConditionRequestv1 `json:"conditions" yaml:"conditions"`
}

// UpdateAlertSettings corresponds to `/api/v1/alerts/settings/{alert_setting_id}`
func (c *Client) UpdateAlertSettings(alertSettings AlertSettingsPayload) error {
	path := fmt.Sprintf("%s/%s", pkg.AlertsSettingsPathV1, alertSettings.AlertID)

	jsonPayload, err := json.Marshal(AlertSettingsPayload{Enable: alertSettings.Enable, Channels: alertSettings.Channels})
	_, err = c.Do(http.MethodPut, path, contentTypeJSON, jsonPayload)
	if err != nil {
		return err
	}

	return nil
}

// CreateAlertSettingsCondition corresponds to `/api/v1/alerts/settings/{alert_setting_id}/condition/{condition_id}`
func (c *Client) CreateAlertSettingsCondition(alertID, condition string, channels []string) error {
	path := fmt.Sprintf("%s/%s/conditions", pkg.AlertsSettingsPathV1, alertID)

	jsonPayload, err := json.Marshal(AlertSettingsConditionPayload{Condition: condition, Channels: channels})
	_, err = c.Do(http.MethodPost, path, contentTypeJSON, jsonPayload)
	if err != nil {
		return err
	}

	return nil
}

// SetAlertSettingsConsumerCondition handles both POST to `/api/v1/alert/settings/{alert_setting_id}/conditions` and
// PUT to `/api/v1/alert/settings/{alert_setting_id}/conditions/{condition_id}` that handles Consumer type of alert category payloads
func (c *Client) SetAlertSettingsConsumerCondition(alertID string, conditionID string, consumerAlert ConsumerAlertConditionRequestv1) error {
	jsonPayload, err := json.Marshal(consumerAlert)
	if err != nil {
		return err
	}

	var path string
	if conditionID != "" {
		path = fmt.Sprintf("%s/%s/conditions/%s", pkg.AlertsSettingsPathV1, alertID, conditionID)
		_, err = c.Do(http.MethodPut, path, contentTypeJSON, jsonPayload)
	} else {
		path = fmt.Sprintf("%s/%s/conditions", pkg.AlertsSettingsPathV1, alertID)
		_, err = c.Do(http.MethodPost, path, contentTypeJSON, jsonPayload)
	}

	if err != nil {
		return err
	}

	return nil
}

// SetAlertSettingsProducerCondition handles both POST to `/api/v1/alert/settings/{alert_setting_id}/conditions` and
// PUT to `/api/v1/alert/settings/{alert_setting_id}/conditions/{condition_id}` that handles Producer type of alert category payloads
func (c *Client) SetAlertSettingsProducerCondition(alertID, conditionID, topic string, threshold Threshold, duration string, channels []string) error {
	if channels == nil {
		channels = []string{}
	}

	payload := AlertConditionRequestv1{
		Condition: DataProduced{
			ConnectionName: "kafka",
			DatasetName:    topic,
			Threshold: Threshold{
				Type:     threshold.Type,
				Messages: threshold.Messages,
			},
			Duration: duration,
		},
		Channels: channels,
	}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	var path string
	if conditionID != "" {
		path = fmt.Sprintf("%s/%s/conditions/%s", pkg.AlertsSettingsPathV1, alertID, conditionID)
		_, err = c.Do(http.MethodPut, path, contentTypeJSON, jsonPayload)
	} else {
		path = fmt.Sprintf("%s/%s/conditions", pkg.AlertsSettingsPathV1, alertID)
		_, err = c.Do(http.MethodPost, path, contentTypeJSON, jsonPayload)
	}

	if err != nil {
		return err
	}

	return nil
}

// GetAlerts returns the registered alerts.
func (c *Client) GetAlerts(pageSize int) (alerts []Alert, err error) {
	path := fmt.Sprintf("%s?pageSize=%d", pkg.AlertEventsPath, pageSize)

	var results AlertResult
	resp, respErr := c.Do(http.MethodGet, path, "", nil)
	if respErr != nil {
		err = respErr
		return
	}

	err = c.ReadJSON(resp, &results)
	alerts = results.Alerts

	return
}

// DeleteAlertEvents deletes alert events.
//
// Deletes all the alert events older than timestamp.
func (c *Client) DeleteAlertEvents(timestamp int64) (err error) {
	queryString := fmt.Sprintf("%s?timestamp=%d", pkg.AlertEventsPath, timestamp)
	resp, err := c.Do(http.MethodDelete, queryString, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

// HasAlertSettingsV2Endpoints answers the question: are the v2 endpoints
// available or not? They were introduced in v5.2, dropping the v1. It returns
// true if a GET /api/v2/alert/settings yields 2xx; it returns false if the
// endpoint yields 404; it returns an error otherwise.
func (c *Client) HasAlertSettingsV2Endpoints() (bool, error) {
	resp, err := c.Do(http.MethodGet, "api/v2/alert/settings", "", nil)
	if err == nil {
		defer resp.Body.Close()
		return true, nil
	}

	var x ResourceError
	if errors.As(err, &x); x.StatusCode == http.StatusNotFound {
		return false, nil
	}
	return false, err
}

// GetAlertSettingsV1 returns all the configured alert settings.
// Alerts are divided into two categories:
//
// * Infrastructure - These are out of the box alerts that be toggled on and offset.
// * Consumer group - These are user-defined alerts on consumer groups.
//
// Alert notifications are the result of an `AlertSetting` Condition being met on an `AlertSetting`.
func (c *Client) GetAlertSettingsV1() (AlertSettings, error) {
	resp, err := c.Do(http.MethodGet, pkg.AlertsSettingsPathV1, "", nil)
	if err != nil {
		return AlertSettings{}, err
	}
	defer resp.Body.Close()

	var settings AlertSettings
	err = c.ReadJSON(resp, &settings)
	return settings, err
}

func (c *Client) GetAlertSettingsV2() (CategorisedAlertRules, error) {
	resp, err := c.Do(http.MethodGet, "api/v2/alert/settings", "", nil)
	if err != nil {
		return CategorisedAlertRules{}, err
	}
	defer resp.Body.Close()

	var categories CategorisedAlertRules
	if err := c.ReadJSON(resp, &categories); err != nil {
		return categories, fmt.Errorf("unmarshal categories: %w", err)
	}
	return categories, nil
}

// GetAlertSettingV2 lists alert settings using GetAlertSettingsV2 and picks
// the alert with requested id from the list.
func (c *Client) GetAlertSettingV2(id int) (AlertRuleV2, error) {
	// There is no endpoint to retrieve individual alerts, hence list them and
	// search for the requested one.
	cat, err := c.GetAlertSettingsV2()
	if err != nil {
		return AlertRuleV2{}, err
	}

	for _, c := range cat.Categories {
		for _, a := range c {
			if a.ID == id {
				return a, nil
			}
		}
	}
	return AlertRuleV2{}, fmt.Errorf("no alert found with id: %d", id)
}

// GetAlertSettingV1 returns a specific alert setting based on its "id".
func (c *Client) GetAlertSettingV1(id int) (setting AlertSettingV1, err error) {
	resp, respErr := c.GetAlertSettingsV1()
	if respErr != nil {
		err = respErr
		return
	}

	for _, category := range resp.Categories.allCategories() {
		for _, v := range category {
			if v.ID == id {
				setting = v
				return
			}
		}
	}

	return
}

// EnableAlertSetting enables a specific alert setting based on its "id".
func (c *Client) EnableAlertSetting(id int, enable bool) error {
	return c.UpdateAlertSettings(AlertSettingsPayload{AlertID: strconv.Itoa(id), Enable: enable, Channels: []string{}})
}

// AlertSettingCondition - used to represent alert settings,
//
//	`ConditionDsl` is generic to handle both "Consumer lag" and "Data Produced" rules
type AlertSettingCondition struct {
	ID           string                 `json:"id,omitempty" header:"ID,text"`
	ConditionDsl map[string]interface{} `json:"conditionDsl" header:"conditionDsl,text"`
	Channels     []string               `json:"channels" header:"channels,text"`
}

// GetAlertSettingConditions returns alert setting's conditions as an array of `AlertSettingCondition`
func (c *Client) GetAlertSettingConditions(id int) ([]AlertSettingCondition, error) {
	conditions := make([]AlertSettingCondition, 0)

	resp, err := c.GetAlertSettingV1(id)
	if err != nil {
		return conditions, err
	}

	for id, details := range resp.ConditionDetails {
		channels := make([]string, 0)
		for _, ch := range details.Channels {
			channels = append(channels, ch.Name)
		}

		conditionDslFlattened := make(map[string]interface{})
		flatten("", details.ConditionDsl, conditionDslFlattened)

		d := AlertSettingCondition{
			ID:           id,
			ConditionDsl: conditionDslFlattened,
			Channels:     channels,
		}
		conditions = append(conditions, d)
	}

	return conditions, err
}

// DeleteAlertSettingCondition deletes a condition from an alert setting.
func (c *Client) DeleteAlertSettingCondition(alertSettingID int, conditionUUID string) error {
	path := fmt.Sprintf("%s/%d/conditions/%s", pkg.AlertsSettingsPathV1, alertSettingID, conditionUUID)
	resp, err := c.Do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}

func flatten(prefix string, src map[string]interface{}, dest map[string]interface{}) {
	if len(prefix) > 0 {
		prefix += "."
	}
	for k, v := range src {
		switch child := v.(type) {
		case map[string]interface{}:
			flatten(prefix+k, child, dest)
		case []interface{}:
			for i := 0; i < len(child); i++ {
				dest[prefix+k+"."+strconv.Itoa(i)] = child[i]
			}
		default:
			dest[prefix+k] = v
		}
	}
}
