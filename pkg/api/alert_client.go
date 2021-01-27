package api

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"

	"github.com/lensesio/lenses-go/pkg"
)

// AlertChannelPayload describes a channel of an alert payload for create/update
type AlertChannelPayload struct {
	Name           string `json:"name" yaml:"name"`
	ConnectionName string `json:"connectionName" yaml:"connectionName"`
	TemplateName   string `json:"templateName" yaml:"templateName"`
	Properties     []KV   `json:"properties" yaml:"properties"`
}

// AlertChannel describes a channel of an alert
type AlertChannel struct {
	ID             string `json:"id" yaml:"id" header:"Id,text"`
	Name           string `json:"name" yaml:"name" header:"Name,text"`
	ConnectionName string `json:"connectionName" yaml:"connectionName" header:"Connection Name,text"`
	TemplateName   string `json:"templateName" yaml:"templateName" header:"Template,text"`
	Properties     []KV   `json:"properties" yaml:"properties" header:"Properties,count"`
	CreatedAt      string `json:"createdAt" yaml:"createdAt"`
	CreatedBy      string `json:"createdBy" yaml:"createdBy"`
	UpdatedAt      string `json:"updatedAt" yaml:"updatedAt"`
	UpdatedBy      string `json:"updatedBy" yaml:"updatedBy"`
}

// AlertChannelWithDetails describes a channel of an alert with more details
type AlertChannelWithDetails struct {
	ID             string `json:"id" yaml:"id" header:"Id,text"`
	Name           string `json:"name" yaml:"name" header:"Name,text"`
	ConnectionName string `json:"connectionName" yaml:"connectionName" header:"Connection Name,text"`
	TemplateName   string `json:"templateName" yaml:"templateName" header:"Template,text"`
	Properties     []KV   `json:"properties" yaml:"properties" header:"Properties"`
	CreatedAt      string `json:"createdAt" yaml:"createdAt" header:"Created at,date"`
	CreatedBy      string `json:"createdBy" yaml:"createdBy" header:"Created by,text"`
	UpdatedAt      string `json:"updatedAt" yaml:"updatedAt" header:"Updated at,date"`
	UpdatedBy      string `json:"updatedBy" yaml:"updatedBy" header:"Updated by,text"`
}

// AlertChannelResponse response for alert channels
type AlertChannelResponse struct {
	PagesAmount int            `json:"pagesAmount" yaml:"pagesAmount" header:"Pages,text"`
	TotalCount  int            `json:"totalCount" yaml:"totalCount" header:"Total,text"`
	Values      []AlertChannel `json:"values" yaml:"values" header:"Values,inline"`
}

// AlertChannelResponseWithDetails response for alert channels
type AlertChannelResponseWithDetails struct {
	PagesAmount int                       `json:"pagesAmount" yaml:"pagesAmount" header:"Pages,text"`
	TotalCount  int                       `json:"totalCount" yaml:"totalCount" header:"Total,text"`
	Values      []AlertChannelWithDetails `json:"values" yaml:"values" header:"Values,inline"`
}

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

//ProducerAlertSettings is the struct used for importing/exporting alert settings
type ProducerAlertSettings struct {
	ID               int                       `json:"alert" yaml:"alert"`
	Description      string                    `json:"description" yaml:"description"`
	ConditionDetails []AlertConditionRequestv1 `json:"conditions" yaml:"conditions"`
}

func constructQueryString(page int, pageSize int, sortField, sortOrder, templateName, channelName string) (query string) {
	v := url.Values{}
	v.Add("pageSize", strconv.Itoa(pageSize))

	if page != 0 {
		v.Add("page", strconv.Itoa(page))
	}
	if sortField != "" {
		v.Add("sortField", sortField)
	}
	if sortOrder != "" {
		v.Add("sortOrder", sortOrder)
	}
	if templateName != "" {
		v.Add("templateName", templateName)
	}
	if channelName != "" {
		v.Add("channelName", channelName)
	}

	query = fmt.Sprintf("%s?%s", pkg.AlertChannelsPath, v.Encode())
	return
}

// GetAlertChannels handles the API call get the list of alert channels
func (c *Client) GetAlertChannels(page int, pageSize int, sortField, sortOrder, templateName, channelName string) (response AlertChannelResponse, err error) {
	path := constructQueryString(page, pageSize, sortField, sortOrder, templateName, channelName)
	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.ReadJSON(resp, &response)
	return
}

// GetAlertChannelsWithDetails handles the API call get the list of alert channels with details
func (c *Client) GetAlertChannelsWithDetails(page int, pageSize int, sortField, sortOrder, templateName, channelName string) (response AlertChannelResponseWithDetails, err error) {
	path := constructQueryString(page, pageSize, sortField, sortOrder, templateName, channelName)
	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.ReadJSON(resp, &response)
	return
}

// DeleteAlertChannel handles the deletion of a channel
func (c *Client) DeleteAlertChannel(channelID string) error {
	path := fmt.Sprintf("%s/%s", pkg.AlertChannelsPath, channelID)
	resp, err := c.Do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// CreateAlertChannel handles the creation of a channel
func (c *Client) CreateAlertChannel(chnl AlertChannelPayload) error {
	var channel = AlertChannelPayload{
		Name:           chnl.Name,
		ConnectionName: chnl.ConnectionName,
		TemplateName:   chnl.TemplateName,
		Properties:     chnl.Properties,
	}
	payload, err := json.Marshal(channel)
	if err != nil {
		return err
	}
	resp, err := c.Do(http.MethodPost, pkg.AlertChannelsPath, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// UpdateAlertChannel handles...take a guess
func (c *Client) UpdateAlertChannel(chnl AlertChannelPayload, channelID string) error {

	var channel = AlertChannelPayload{
		Name:           chnl.Name,
		ConnectionName: chnl.ConnectionName,
		TemplateName:   chnl.TemplateName,
		Properties:     chnl.Properties,
	}
	path := fmt.Sprintf("%s/%s", pkg.AlertChannelsPath, channelID)
	payload, err := json.Marshal(channel)
	if err != nil {
		return err
	}
	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// UpdateAlertSettings corresponds to `/api/v1/alerts/settings/{alert_setting_id}`
func (c *Client) UpdateAlertSettings(alertSettings AlertSettingsPayload) error {
	path := fmt.Sprintf("%s/%s", pkg.AlertsSettingsPath, alertSettings.AlertID)

	jsonPayload, err := json.Marshal(AlertSettingsPayload{Enable: alertSettings.Enable, Channels: alertSettings.Channels})
	_, err = c.Do(http.MethodPut, path, contentTypeJSON, jsonPayload)

	if err != nil {
		return err
	}

	return nil
}

// UpdateAlertSettingsCondition corresponds to `/api/v1/alerts/settings/{alert_setting_id}/condition/{condition_id}`
func (c *Client) UpdateAlertSettingsCondition(alertID, condition, conditionID string, channels []string) error {
	path := fmt.Sprintf("%s/%s/conditions/%s", pkg.AlertsSettingsPath, alertID, conditionID)

	jsonPayload, err := json.Marshal(AlertSettingsConditionPayload{Condition: condition, Channels: channels})
	_, err = c.Do(http.MethodPut, path, contentTypeJSON, jsonPayload)

	if err != nil {
		return err
	}

	return nil
}

// CreateAlertSettingsCondition corresponds to `/api/v1/alerts/settings/{alert_setting_id}/condition/{condition_id}`
func (c *Client) CreateAlertSettingsCondition(alertID, condition string, channels []string) error {
	path := fmt.Sprintf("%s/%s/conditions", pkg.AlertsSettingsPath, alertID)

	jsonPayload, err := json.Marshal(AlertSettingsConditionPayload{Condition: condition, Channels: channels})
	_, err = c.Do(http.MethodPost, path, contentTypeJSON, jsonPayload)

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
		path = fmt.Sprintf("%s/%s/conditions/%s", pkg.AlertsSettingsPath, alertID, conditionID)
		_, err = c.Do(http.MethodPut, path, contentTypeJSON, jsonPayload)
	} else {
		path = fmt.Sprintf("%s/%s/conditions", pkg.AlertsSettingsPath, alertID)
		_, err = c.Do(http.MethodPost, path, contentTypeJSON, jsonPayload)
	}

	if err != nil {
		return err
	}

	return nil
}

// AlertHandler is the type of func that can be registered to receive alerts via the `GetAlertsLive`.
type AlertHandler func(Alert) error

// GetAlertsLive receives alert notifications in real-time from the server via a Send Server Event endpoint.
func (c *Client) GetAlertsLive(handler AlertHandler) error {
	resp, err := c.Do(http.MethodGet, pkg.AlertsPathSSE, contentTypeJSON, nil, func(r *http.Request) error {
		r.Header.Add(acceptHeaderKey, "application/json, text/event-stream")
		return nil
	}, schemaAPIOption)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	reader, err := c.acquireResponseBodyStream(resp)
	if err != nil {
		return err
	}

	streamReader := bufio.NewReader(reader)

	for {
		line, err := streamReader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				return nil // we read until the the end, exit with no error here.
			}
			return err // exit on first failure.
		}

		if len(line) < shiftN+1 { // even more +1 for the actual event.
			// almost empty or totally invalid line,
			// empty message maybe,
			// we don't care, we ignore them at any way.
			continue
		}

		if !bytes.HasPrefix(line, dataPrefix) {
			return fmt.Errorf("client: see: fail to read the event, the incoming message has no [%s] prefix", string(dataPrefix))
		}

		message := line[shiftN:] // we need everything after the 'data:'.

		if len(message) < 2 {
			continue // do NOT stop here, let the connection active.
		}

		alert := Alert{}

		if err = json.Unmarshal(message, &alert); err != nil {
			// exit on first error here as well.
			return err
		}

		if err = handler(alert); err != nil {
			return err // stop on first error by the caller.
		}
	}
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

// GetAlertSettings returns all the configured alert settings.
// Alerts are divided into two categories:
//
// * Infrastructure - These are out of the box alerts that be toggled on and offset.
// * Consumer group - These are user-defined alerts on consumer groups.
//
// Alert notifications are the result of an `AlertSetting` Condition being met on an `AlertSetting`.
func (c *Client) GetAlertSettings() (AlertSettings, error) {
	resp, err := c.Do(http.MethodGet, pkg.AlertsSettingsPath, "", nil)
	if err != nil {
		return AlertSettings{}, err
	}

	var settings AlertSettings
	err = c.ReadJSON(resp, &settings)
	return settings, err
}

// GetAlertSetting returns a specific alert setting based on its "id".
func (c *Client) GetAlertSetting(id int) (setting AlertSetting, err error) {
	resp, respErr := c.GetAlertSettings()
	if respErr != nil {
		err = respErr
		return
	}

	for _, v := range resp.Categories.Consumers {
		if v.ID == id {
			setting = v
			return
		}
	}

	for _, v := range resp.Categories.Infrastructure {
		if v.ID == id {
			setting = v
			return
		}
	}

	for _, v := range resp.Categories.Producers {
		if v.ID == id {
			setting = v
			return
		}
	}

	return
}

// EnableAlertSetting enables a specific alert setting based on its "id".
func (c *Client) EnableAlertSetting(id int, enable bool) error {
	return c.UpdateAlertSettings(AlertSettingsPayload{AlertID: strconv.Itoa(id), Enable: enable, Channels: []string{}})
}

// AlertSettingConditions map with UUID as key and the condition as value, used on `GetAlertSettingConditions`.
type AlertSettingConditions map[string]string

// GetAlertSettingConditions returns alert setting's conditions as a map of strings.
func (c *Client) GetAlertSettingConditions(id int) (AlertSettingConditions, error) {
	resp, err := c.GetAlertSetting(id)
	if err != nil {
		return AlertSettingConditions{}, err
	}
	return resp.Conditions, err
}

// DeleteAlertSettingCondition deletes a condition from an alert setting.
func (c *Client) DeleteAlertSettingCondition(alertSettingID int, conditionUUID string) error {
	path := fmt.Sprintf("%s/%d/conditions/%s", pkg.AlertsSettingsPath, alertSettingID, conditionUUID)
	resp, err := c.Do(http.MethodDelete, path, "", nil)
	if err != nil {
		return err
	}

	return resp.Body.Close()
}
