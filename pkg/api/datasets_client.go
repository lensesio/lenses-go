package api

import (
	"fmt"
	"net/http"
	"encoding/json"

	"github.com/lensesio/lenses-go/pkg"
)

// UpdateDatasetsMetadata Struct
type UpdateDatasetsMetadata struct {
	Description    string    `json:"description" yaml:"description"`
}

// updateDatasetsMetadata validates and creates new dataset metadata json payload
func updateDatasetMetadata(description string) (jsonPayload []byte, err error) {
	payload := UpdateDatasetsMetadata{
		Description:  description,
	}

	jsonPayload, err = json.Marshal(payload)

	return
}

// UpdateMetadata Method
func (c *Client) UpdateMetadata(connection, name, description string) ( err error) {
	if connection == "" {
		err = errRequired("Required argument --connection not given")
		return
	}

	if name == "" {
		err = errRequired("Required argument --name not given")
		return
	}

	jsonPayload, err := updateDatasetMetadata(description)
	if err != nil {
		return
	}

	path := fmt.Sprintf("api/%s/%s/%s", pkg.DatasetsAPIPath, connection, name)

	resp, err := c.Do(http.MethodPatch, path, contentTypeJSON, jsonPayload)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return
}