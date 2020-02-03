package api

import (
	"fmt"
	"net/http"

	"github.com/landoop/lenses-go/pkg"
)

// Connection Templates API

// ConnectionTemplateMetadata type
type ConnectionTemplateMetadata struct {
	Author      string `json:"author" yaml:"author" header:"Author,text"`
	Description string `json:"description" yaml:"description" header:"Description,text"`
	DocURL      string `json:"docUrl" yaml:"docUrl" header:"Doc Url,text"`
	GitRepo     string `json:"gitRepo" yaml:"gitRepo" header:"Git Repo,text"`
	GitCommit   string `json:"gitCommit" yaml:"gitCommit" header:"Git Commit,text"`
	Image       string `json:"image" yaml:"image" header:"Image,text"`
	ImageTag    string `json:"imageTag" yaml:"imageTag" header:"Image Tag,text"`
}

// ConnectionTemplateConfigType type
type ConnectionTemplateConfigType struct {
	Name        string `json:"name" yaml:"name" header:"Name,text"`
	DisplayName string `json:"displayName" yaml:"DisplayName" header:"Display Name,text"`
}

// ConnectionTemplateConfig type
type ConnectionTemplateConfig struct {
	Key         string                       `json:"key" yaml:"key" header:"key,text"`
	DisplayName string                       `json:"displayName" yaml:"displayName" header:"Display Name,text"`
	Placeholder string                       `json:"placeholder" yaml:"placeholder" header:"Placeholder,text"`
	Description string                       `json:"description" yaml:"description" header:"Description,text"`
	Required    bool                         `json:"required" yaml:"required" header:"Required,text"`
	Mounted     bool                         `json:"mounted" yaml:"mounted" header:"Mounted,text"`
	Type        ConnectionTemplateConfigType `json:"type" yaml:"type" header:"Type,text"`
}

// ConnectionTemplate type
type ConnectionTemplate struct {
	Name     string                     `json:"	name" yaml:"name" header:"Name,text"`
	Version  string                     `json:"version" yaml:"version" header:"Version,text"`
	BuiltIn  bool                       `json:"builtIn" yaml:"buildIn" header:"BuiltIn,text"`
	Enabled  bool                       `json:"enabled" yaml:"enabled" header:"Enabled,text"`
	Category string                     `json:"category" yaml:"category"`
	Type     string                     `json:"type" yaml:"type"`
	Metadata ConnectionTemplateMetadata `json:"metadata" yaml:"metadata"`
	Config   []ConnectionTemplateConfig `json:"configuration" yaml:"configuration"`
}

// GetConnectionTemplates returns all connections
func (c *Client) GetConnectionTemplates() (response []ConnectionTemplate, err error) {
	path := fmt.Sprintf("api/%s", pkg.ConnectionTemplatesAPIPath)

	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}

	if err = c.ReadJSON(resp, &response); err != nil {
		return
	}

	return
}
