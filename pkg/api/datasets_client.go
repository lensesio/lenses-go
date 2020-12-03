package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/lensesio/lenses-go/pkg"
)

// UpdateDatasetDescription Struct
type UpdateDatasetDescription struct {
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// updateDatasetsDescrciption creates new dataset metadata json payload
func updateDatasetDescription(description string) (jsonPayload []byte, err error) {
	payload := UpdateDatasetDescription{
		Description: description,
	}

	jsonPayload, err = json.Marshal(payload)

	return
}

// UpdateDatasetDescription validates that the supplied parameters are not empty
// note: we intenionally allow here description to be empty as that is needed in order to remove it
func (c *Client) UpdateDatasetDescription(connection, name, description string) (err error) {
	if len(strings.TrimSpace(connection)) == 0 {
		return errors.New("Required argument --connection not given or blank")
	}

	if len(strings.TrimSpace(name)) == 0 {
		return errors.New("Required argument --name not given or blank")
	}

	jsonPayload, err := updateDatasetDescription(description)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("api/%s/%s/%s/description", pkg.DatasetsAPIPath, connection, name)

	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, jsonPayload)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}
