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

// UpdateDatasetTags struct
type UpdateDatasetTags struct {
	Tags []DatasetTag `json:"tags" yaml:"tags"`
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

// UpdateDatasetTags sets the dataset tags from the supplied list
func (c *Client) UpdateDatasetTags(connection, name string, tags []string) (err error) {
	if len(strings.TrimSpace(connection)) == 0 {
		return errors.New("Required argument --connection not given or blank")
	}

	if len(strings.TrimSpace(name)) == 0 {
		return errors.New("Required argument --name not given or blank")
	}

	datasetTags := []DatasetTag{}
	for _, tag := range tags {
		if len(tag) == 0 || len(tag) > 255 {
			return errors.New("tags contain blank characters, or contain strings longer than 256 characters")
		}
		datasetTags = append(datasetTags, DatasetTag{Name: tag})
	}
	payload := UpdateDatasetTags{
		Tags: datasetTags,
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("api/%s/%s/%s/tags", pkg.DatasetsAPIPath, connection, name)

	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, jsonPayload)
	if err != nil {
		return err
	}

	defer resp.Body.Close()
	return nil
}
