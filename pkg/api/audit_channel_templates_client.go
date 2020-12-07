package api

import (
	"net/http"

	"github.com/lensesio/lenses-go/pkg"
)

// GetAuditChannelTemplates returns all audit channel templates
func (c *Client) GetAuditChannelTemplates() (response []ChannelTemplate, err error) {
	resp, err := c.Do(http.MethodGet, pkg.AuditChannelTemplatesPath, contentTypeJSON, nil)
	if err != nil {
		return
	}

	if err = c.ReadJSON(resp, &response); err != nil {
		return
	}

	return
}
