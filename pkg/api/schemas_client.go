package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"
)

//Tags Struct
type Tags struct {
	Name string `json:"name"`
}

//Version Struct
type Version struct {
	ID      int    `json:"id"`
	Version int    `json:"version"`
	Schema  string `json:"schema"`
	Format  string `json:"format"`
}

//GetSchemaRes Struct
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

//DatasetsResp struct maps to the `api/v1/datasets` response payload
type DatasetsResp struct {
	Datasets struct {
		Values []struct {
			Name          string `json:"name"`
			Format        string `json:"format"`
			Version       int    `json:"version"`
			Compatibility string `json:"compatibility"`
		} `json:"values"`
		PagesAmount int `json:"pagesAmount"`
		TotalCount  int `json:"totalCount"`
	} `json:"datasets"`
	SourceTypes []string `json:"sourceTypes"`
}

//Subjects struct is used at 'schema-registy subjects' cmd
type Subjects []struct {
	Name          string `json:"name" yaml:"name" header:"name"`
	Format        string `json:"format" yaml:"format" header:"format"`
	Version       int    `json:"version" yaml:"version" header:"latest version"`
	Compatibility string `json:"compatibility" yaml:"compatibility" header:"compatibility"`
}

//GetSubjects retrieves all registered subjects
func (c *Client) GetSubjects() (subs Subjects, err error) {

	resp, err := c.Do(http.MethodGet, "api/v1/datasets?pageSize=99999&connections=schema-registry", "gzip", nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var datasets DatasetsResp
	if err = c.ReadJSON(resp, &datasets); err != nil {
		return
	}

	return (Subjects)(datasets.Datasets.Values), nil
}

//GetSchema returns the details of a schema
func (c *Client) GetSchema(name string) (response GetSchemaRes, err error) {
	const basePath = "api/v1/datasets/schema-registry"
	path := fmt.Sprintf("%s/%s", basePath, name)

	if name == "" {
		err = fmt.Errorf("name is required")
		return
	}

	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)

	if err != nil {
		return
	}

	defer resp.Body.Close()

	err = c.ReadJSON(resp, &response)
	if err != nil {
		return
	}

	return
}

//WriteSchemaReq Struct
type WriteSchemaReq struct {
	Format string `json:"format"`
	Schema string `json:"schema"`
}

//WriteSchema creates a schema if it doens't exist, updates it otherwise
func (c *Client) WriteSchema(name string, request WriteSchemaReq) (err error) {
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

	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return
}

//SetSchemaCompatibilityReq Struct
type SetSchemaCompatibilityReq struct {
	Compatibility string `json:"compatibility"`
}

//SetSchemaCompatibility set the compatibility of a schema
func (c *Client) SetSchemaCompatibility(name string, request SetSchemaCompatibilityReq) (err error) {
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

	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return
}

//SetGlobalCompatibilityReq Struct
type SetGlobalCompatibilityReq struct {
	Compatibility string `json:"compatibility"`
}

//SetGlobalCompatibility set the default compatibility of the schema registry
func (c *Client) SetGlobalCompatibility(request SetGlobalCompatibilityReq) (err error) {
	const basePath = "api/v1/sr/default/config"

	if request.Compatibility == "" {
		return fmt.Errorf("compatibility is required")
	}

	payload, err := json.Marshal(request)
	if err != nil {
		return
	}

	resp, err := c.Do(http.MethodPut, basePath, contentTypeJSON, payload)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return
}

//RemoveSchemaVersion removes a particular schema version
func (c *Client) RemoveSchemaVersion(name string, version string) (err error) {
	const basePath = "api/v1/sr/default/subject"
	path := fmt.Sprintf("%s/%s/version/%s", basePath, name, version)

	if name == "" {
		return fmt.Errorf("name is required")
	}

	if version == "" {
		return fmt.Errorf("version is required")
	}

	resp, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)

	if err != nil {
		return err
	}

	defer resp.Body.Close()

	return
}

//RemoveSchema removes the schema and all its versions
func (c *Client) RemoveSchema(name string) (err error) {
	const basePath = "api/v1/sr/default/subject"
	path := fmt.Sprintf("%s/%s", basePath, name)

	if name == "" {
		return fmt.Errorf("name is required")
	}

	resp, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return
}
