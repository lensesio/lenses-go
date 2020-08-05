package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lensesio/lenses-go/pkg"
)

// Connections API

// ConnectionApp type
type ConnectionApp struct {
	Name   string `json:"name" yaml:"name" header:"Name,text"`
	Status string `json:"status" yaml:"status" header:"Status,text"`
}

// ConnectionList type
type ConnectionList struct {
	Name            string   `json:"name" yaml:"name" header:"Name,text"`
	TemplateName    string   `json:"templateName" yaml:"templateName" header:"Template Name,text"`
	TemplateVersion int      `json:"templateVersion" yaml:"templateVersion" header:"Template Version,int"`
	Tags            []string `json:"tags" yaml:"tags" header:"Tags,text"`
	ReadOnly        bool     `json:"readOnly" yaml:"readOnly" header:"Read only"`
}

// Connection type
type Connection struct {
	Name            string             `json:"name" yaml:"name" header:"Name,text"`
	TemplateName    string             `json:"templateName" yaml:"templateName" header:"Template Name,text"`
	TemplateVersion int                `json:"templateVersion" yaml:"templateVersion" header:"Template Version,int"`
	BuiltIn         bool               `json:"builtIn" yaml:"builtIn" header:"BuiltIn,text"`
	ReadOnly        bool               `json:"readOnly" yaml:"readOnly" header:"Read only"`
	Configuration   []ConnectionConfig `json:"configuration" yaml:"configuration"`
	CreatedBy       string             `json:"createdBy" yaml:"createdBy" header:"Created By,text"`
	CreatedAt       int64              `json:"createdAt" yaml:"createdAt" header:"Created At,text"`
	ModifiedBy      string             `json:"modifiedBy" yaml:"modifiedBy" header:"Modified By,text"`
	ModifiedAt      int64              `json:"modifiedAt" yaml:"modifiedAt" header:"Modified At,text"`
	Tags            []string           `json:"tags" yaml:"tags" header:"Tags,text"`
}

// GetConnections returns all connections
func (c *Client) GetConnections() (response []ConnectionList, err error) {
	path := fmt.Sprintf("api/%s", pkg.ConnectionsAPIPath)

	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}

	if err = c.ReadJSON(resp, &response); err != nil {
		return
	}

	return
}

// GetConnection returns a specific connection
func (c *Client) GetConnection(name string) (response Connection, err error) {
	path := fmt.Sprintf("api/%s/%s", pkg.ConnectionsAPIPath, name)

	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}

	if err = c.ReadJSON(resp, &response); err != nil {
		return
	}

	return
}

// ConnectionConfig type
type ConnectionConfig struct {
	Key   string      `json:"key" yaml:"key"`
	Value interface{} `json:"value" yaml:"value"`
}

// CreateConnectionPayload type
type CreateConnectionPayload struct {
	Name          string             `json:"name" yaml:"name"`
	TemplateName  string             `json:"templateName" yaml:"templateName"`
	Configuration []ConnectionConfig `json:"configuration" yaml:"configuration"`
	Tags          []string           `json:"tags" yaml:"tags"`
}

// parseConnectionConfigurationValues parses the config values given as a JSON array string
func parseConnectionConfigurationValues(configString string) (configKeyValueArray []ConnectionConfig, err error) {
	if len(configString) == 0 {
		return configKeyValueArray, fmt.Errorf("Configuration values not found")
	}

	// map that holds the decoded JSON configurations
	var jsonConfig []map[string]interface{}
	// array that holds config options (can be of any type)
	configByte := []byte(configString)
	if err = json.Unmarshal(configByte, &jsonConfig); err != nil {
		return
	}
	configKeyValueArray = make([]ConnectionConfig, len(jsonConfig))

	for i := range jsonConfig {
		configKeyValueArray[i] = ConnectionConfig{jsonConfig[i]["key"].(string), jsonConfig[i]["value"]}
	}

	return
}

// createConnectionPayload validates and creates new connection json payload
func createConnectionPayload(connectionName string, templateName string, configKeyValueArray []ConnectionConfig, tags []string) (jsonPayload []byte, err error) {
	// required parameters
	if connectionName == "" || templateName == "" || len(configKeyValueArray) == 0 {
		err = fmt.Errorf("client: required argument missing")
		return
	}

	payload := CreateConnectionPayload{
		Name:          connectionName,
		TemplateName:  templateName,
		Configuration: configKeyValueArray,
		Tags:          tags,
	}

	jsonPayload, err = json.Marshal(payload)

	return
}

// UpdateConnectionPayload type
type UpdateConnectionPayload struct {
	Name          string             `json:"name" yaml:"name"`
	Configuration []ConnectionConfig `json:"configuration" yaml:"configuration"`
	Tags          []string           `json:"tags" yaml:"tags"`
}

// updateConnectionPayload validates and creates new connection json payload
func updateConnectionPayload(connectionName string, configKeyValueArray []ConnectionConfig, tags []string) (jsonPayload []byte, err error) {
	// required parameters
	if len(configKeyValueArray) == 0 {
		err = errRequired("Required argument config not given")
		return
	}

	payload := UpdateConnectionPayload{
		Name:          connectionName,
		Configuration: configKeyValueArray,
		Tags:          tags,
	}

	jsonPayload, err = json.Marshal(payload)

	return
}

// CreateConnection creates a new Lenses connection
func (c *Client) CreateConnection(connectionName string, templateName string, configString string, configArray []ConnectionConfig, tags []string) (err error) {
	if len(configString) != 0 {
		if configArray, err = parseConnectionConfigurationValues(configString); err != nil {
			return
		}
	}

	jsonPayload, err := createConnectionPayload(connectionName, templateName, configArray, tags)
	if err != nil {
		return
	}

	path := fmt.Sprintf("api/%s", pkg.ConnectionsAPIPath)

	resp, err := c.Do(http.MethodPost, path, contentTypeJSON, jsonPayload)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return
}

// UpdateConnection updates a Lenses connection
func (c *Client) UpdateConnection(connectionName string, newName string, configString string, configArray []ConnectionConfig, tags []string) (err error) {
	if connectionName == "" {
		return errRequired("Required argument --connectionName not given")
	}

	if len(configString) != 0 {
		if configArray, err = parseConnectionConfigurationValues(configString); err != nil {
			return
		}
	}

	jsonPayload, err := updateConnectionPayload(newName, configArray, tags)
	if err != nil {
		return
	}

	path := fmt.Sprintf("api/%s/%s", pkg.ConnectionsAPIPath, connectionName)

	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, jsonPayload)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return
}

// DeleteConnection deletes a new Lenses connection
func (c *Client) DeleteConnection(connectionName string) (err error) {
	if connectionName == "" {
		return errRequired("Required argument connectionName not given")
	}

	path := fmt.Sprintf("api/%s/%s", pkg.ConnectionsAPIPath, connectionName)

	resp, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return
}
