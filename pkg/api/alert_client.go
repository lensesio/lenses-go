package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/landoop/lenses-go/pkg"
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
	AlertID     string   `json:"alert" yaml:"alert"`
	ConditionID string   `json:"conditionID,omitempty" yaml:"conditionID"`
	Condition   string   `json:"condition" yaml:"condition"`
	Channels    []string `json:"channels" yaml:"channels"`
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
