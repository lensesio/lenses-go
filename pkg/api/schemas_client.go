package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

type Tags struct {
	Name string `json:"name"`
}

type Version struct {
	ID      int    `json:"id"`
	Version int    `json:"version"`
	Schema  string `json:"schema"`
	Format  string `json:"format"`
}

type GetSchemaRes struct {
	Name           string   `json:"name"`
	ConnectionName string   `json:"connectionName"`
	IsSystemEntity bool     `json:"isSystemEntity"`
	Permissions    []string `json:"permissions"`
	Description    string   `json:"description"`
	Tags           []Tags   `json:"tags"`
	Format         string   `json:"format"`
	Schema         string   `json:"schema"`
	Version        int      `json:"version"`
	SchemaID       int      `json:"schemaId"`
	Compatibility  string   `json:"compatibility"`
	SourceType     string   `json:"sourceType"`
}

type GetSchemaReq struct {
	Name string `json:"name"`
}

func (c *Client) GetV2Schema(name string) (response GetSchemaRes, err error) {
	const basePath = "api/v1/datasets/lenses-default-schema-registry"
	path := fmt.Sprintf("%s/%s", basePath, name)

	if name == "" {
		err = fmt.Errorf("name is required")
		return
	}

	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)

	if err != nil {
		return
	}

	err = c.ReadJSON(resp, &response)
	return
}

type WriteSchemaReq struct {
	Format string `json:"format"`
	Schema string `json:"schema"`
}

func (c *Client) WriteSchema(name string, request WriteSchemaReq) error {
	const basePath = "api/v1/sr/default/subject"
	path := fmt.Sprintf("%s/%s/current-version", basePath, name)

	if name == "" {
		return fmt.Errorf("name is required")
	}

	if request.Format == "" {
		return fmt.Errorf("format is required")
	}

	if request.Schema == "" {
		return fmt.Errorf("schema is required")
	}

	payload, err := json.Marshal(request)

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	_, err = c.Do(http.MethodPut, path, contentTypeJSON, payload)
	return err
}

type SetSchemaCompatibilityReq struct {
	Compatibility string `json:"compatibility"`
}

func (c *Client) SetSchemaCompatibility(name string, request SetSchemaCompatibilityReq) error {
	const basePath = "api/v1/sr/default/subject"
	path := fmt.Sprintf("%s/%s/config", basePath, name)

	if name == "" {
		return fmt.Errorf("name is required")
	}

	if request.Compatibility == "" {
		return fmt.Errorf("compatibility is required")
	}

	payload, err := json.Marshal(request)

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	_, err = c.Do(http.MethodPut, path, contentTypeJSON, payload)
	return err
}

type SetGlobalCompatibilityReq struct {
	Compatibility string `json:"compatibility"`
}

func (c *Client) SetGlobalCompatibility(request SetGlobalCompatibilityReq) error {
	const basePath = "api/v1/sr/default/config"

	if request.Compatibility == "" {
		return fmt.Errorf("compatibility is required")
	}

	payload, err := json.Marshal(request)

	if err != nil {
		return errors.Wrap(err, "Request failed")
	}

	_, err = c.Do(http.MethodPut, basePath, contentTypeJSON, payload)
	return err
}

func (c *Client) RemoveSchemaVersion(name string, version string) error {
	const basePath = "api/v1/sr/default/subject"
	path := fmt.Sprintf("%s/%s/version/%s", basePath, name, version)

	if name == "" {
		return fmt.Errorf("name is required")
	}

	if version == "" {
		return fmt.Errorf("version is required")
	}

	_, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)
	return err
}

func (c *Client) RemoveSchema(name string) error {
	const basePath = "api/v1/sr/default/subject"
	path := fmt.Sprintf("%s/%s", basePath, name)

	if name == "" {
		return fmt.Errorf("name is required")
	}

	_, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)
	return err
}
