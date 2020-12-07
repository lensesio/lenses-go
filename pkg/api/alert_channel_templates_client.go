package api

import (
	"net/http"

	"github.com/lensesio/lenses-go/pkg"
)

// ChannelTemplate payload struct used for alert and audit
type ChannelTemplate struct {
	ID       int    `json:"id" yaml:"id"`
	Name     string `json:"name" yaml:"name" header:"name"`
	Version  string `json:"version" yaml:"version" header:"version"`
	Enabled  bool   `json:"enabled" yaml:"enabled" header:"enabled"`
	BuiltIn  bool   `json:"builtIn" yaml:"builtin" header:"builtin"`
	Metadata struct {
		Author      string `json:"author"`
		Description string `json:"description"`
	} `json:"metadata"`
	Configuration []struct {
		ID          int    `json:"id"`
		Key         string `json:"key"`
		DisplayName string `json:"displayName"`
		Placeholder string `json:"placeholder"`
		Description string `json:"description"`
		Type        struct {
			Name        string      `json:"name"`
			DisplayName string      `json:"displayName"`
			EnumValues  interface{} `json:"enumValues"`
		} `json:"type"`
		Required bool `json:"required"`
		Provided bool `json:"provided"`
	} `json:"configuration"`
	SuitableConnections []struct {
		TemplateName string `json:"templateName"`
		Name         string `json:"name"`
	} `json:"suitableConnections"`
}

// GetAlertChannelTemplates returns all alert channel templates
func (c *Client) GetAlertChannelTemplates() (response []ChannelTemplate, err error) {
	resp, err := c.Do(http.MethodGet, pkg.AlertChannelTemplatesPath, contentTypeJSON, nil)
	if err != nil {
		return
	}

	if err = c.ReadJSON(resp, &response); err != nil {
		return
	}

	return
}
