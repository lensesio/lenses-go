package api

import (
	"fmt"
	"net/http"

	"github.com/lensesio/lenses-go/v5/pkg"
)

func (c *Client) GetConnectionsState() (yaml string, err error) {
	resp, err := c.Do(http.MethodGet, pkg.ProvisionedConnectionsPath, contentTypeYaml, nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	bodyBytes, err := c.ReadResponseBody(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	yaml = string(bodyBytes)

	return yaml, nil
}
