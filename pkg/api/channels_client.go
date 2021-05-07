package api

import (
	"net/http"

	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
)

// TODO AC-1458: better comment
// As Alert and Audit channels have the same API contracts,
// some parts of the code might be reused.

// ChannelPayload describes a channel of an alert payload for create/update
type ChannelPayload struct {
	Name           string `json:"name" yaml:"name"`
	ConnectionName string `json:"connectionName" yaml:"connectionName"`
	TemplateName   string `json:"templateName" yaml:"templateName"`
	Properties     []KV   `json:"properties" yaml:"properties"`
}

// Channel describes a channel of an alert
type Channel struct {
	ID              string `json:"id,omitempty" yaml:"id" header:"Id,text"`
	Name            string `json:"name,omitempty" yaml:"name" header:"Name,text"`
	ConnectionName  string `json:"connectionName,omitempty" yaml:"connectionName" header:"Connection Name,text"`
	TemplateName    string `json:"templateName,omitempty" yaml:"templateName" header:"Template,text"`
	TemplateVersion int    `json:"templateVersion,omitempty" yaml:"templateVersion" header:"Template version"`
	Properties      []KV   `json:"properties,omitempty" yaml:"properties" header:"Properties,count"`
	CreatedAt       string `json:"createdAt,omitempty" yaml:"createdAt"`
	CreatedBy       string `json:"createdBy,omitempty" yaml:"createdBy"`
	UpdatedAt       string `json:"updatedAt,omitempty" yaml:"updatedAt"`
	UpdatedBy       string `json:"updatedBy,omitempty" yaml:"updatedBy"`
}

// ChannelWithDetails describes a channel of an alert with more details
type ChannelWithDetails struct {
	ID              string `json:"id" yaml:"id" header:"Id,text"`
	Name            string `json:"name" yaml:"name" header:"Name,text"`
	ConnectionName  string `json:"connectionName" yaml:"connectionName" header:"Connection Name,text"`
	TemplateName    string `json:"templateName" yaml:"templateName" header:"Template,text"`
	TemplateVersion int    `json:"templateVersion,omitempty" yaml:"templateVersion" header:"Template version"`
	Properties      []KV   `json:"properties" yaml:"properties" header:"Properties"`
	CreatedAt       string `json:"createdAt" yaml:"createdAt" header:"Created at,date"`
	CreatedBy       string `json:"createdBy" yaml:"createdBy" header:"Created by,text"`
	UpdatedAt       string `json:"updatedAt" yaml:"updatedAt" header:"Updated at,date"`
	UpdatedBy       string `json:"updatedBy" yaml:"updatedBy" header:"Updated by,text"`
}

// ChannelResponse response for alert channels
type ChannelResponse struct {
	PagesAmount int       `json:"pagesAmount" yaml:"pagesAmount" header:"Pages,text"`
	TotalCount  int       `json:"totalCount" yaml:"totalCount" header:"Total,text"`
	Values      []Channel `json:"values" yaml:"values" header:"Values,inline"`
}

// ChannelResponseWithDetails response for alert channels
type ChannelResponseWithDetails struct {
	PagesAmount int                  `json:"pagesAmount" yaml:"pagesAmount" header:"Pages,text"`
	TotalCount  int                  `json:"totalCount" yaml:"totalCount" header:"Total,text"`
	Values      []ChannelWithDetails `json:"values" yaml:"values" header:"Values,inline"`
}

// GetChannels read channels (can be used both for audit and alert channels)
func (c *Client) GetChannels(path string, page int, pageSize int, sortField, sortOrder, templateName, channelName string) (response ChannelResponse, err error) {
	queryString := constructQueryString(path, page, pageSize, sortField, sortOrder, templateName, channelName)
	resp, err := c.Do(http.MethodGet, queryString, contentTypeJSON, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.ReadJSON(resp, &response)
	return
}

// GetChannelsWithDetails read channels details (can be used both for audit and alert channels)
func (c *Client) GetChannelsWithDetails(path string, page int, pageSize int, sortField, sortOrder, templateName, channelName string) (response ChannelResponseWithDetails, err error) {
	queryString := constructQueryString(path, page, pageSize, sortField, sortOrder, templateName, channelName)
	resp, err := c.Do(http.MethodGet, queryString, contentTypeJSON, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.ReadJSON(resp, &response)
	return
}

// CreateChannel handles the creation of a channel
func (c *Client) CreateChannel(chnl ChannelPayload, channelPath string) error {
	var channel = ChannelPayload{
		Name:           chnl.Name,
		ConnectionName: chnl.ConnectionName,
		TemplateName:   chnl.TemplateName,
		Properties:     chnl.Properties,
	}
	payload, err := json.Marshal(channel)
	if err != nil {
		return err
	}
	resp, err := c.Do(http.MethodPost, channelPath, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// UpdateChannel handles...take a guess
func (c *Client) UpdateChannel(chnl ChannelPayload, channelPath, channelID string) error {

	var channel = ChannelPayload{
		Name:           chnl.Name,
		ConnectionName: chnl.ConnectionName,
		TemplateName:   chnl.TemplateName,
		Properties:     chnl.Properties,
	}
	path := fmt.Sprintf("%s/%s", channelPath, channelID)
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

// DeleteChannel deletes a channel (can be used both for audit and alert channels)
func (c *Client) DeleteChannel(path, channelID string) error {
	queryString := fmt.Sprintf("%s/%s", path, channelID)
	resp, err := c.Do(http.MethodDelete, queryString, "", nil)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

func constructQueryString(path string, page int, pageSize int, sortField, sortOrder, templateName, channelName string) (query string) {
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

	query = fmt.Sprintf("%s?%s", path, v.Encode())
	return
}
