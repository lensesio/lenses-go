package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const groupPath = "api/v1/group"

// Namespace the payload object for namespaces
type Namespace struct {
	Wildcards   []string `json:"wildcards" yaml:"wildcards" header:"Wildcards"`
	Permissions []string `json:"permissions" yaml:"permissions" header:"Permissions"`
	Connection  string   `json:"connection" yaml:"connection" header:"connection"`
}

// Group the payload object
type Group struct {
	Name                       string      `json:"name" yaml:"name" header:"Name"`
	Description                string      `json:"description,omitempty" yaml:"description" header:"Description"`
	Namespaces                 []Namespace `json:"namespaces" yaml:"dataNamespaces" header:"Namespaces,count"`
	ScopedPermissions          []string    `json:"scopedPermissions" yaml:"applicationPermissions" header:"Application Permissions,count"`
	AdminPermissions           []string    `json:"adminPermissions" yaml:"adminPermissions" header:"Admin Permissions,count"`
	UserAccountsCount          int         `json:"userAccounts" yaml:"userAccounts" header:"User Accounts"`
	ServiceAccountsCount       int         `json:"serviceAccounts" yaml:"serviceAccounts" header:"Service Accounts"`
	ConnectClustersPermissions []string    `json:"connectClustersPermissions" yaml:"connectClustersPermissions" header:"Connect clusters access"`
}

// GetGroups returns the list of groups
func (c *Client) GetGroups() (groups []Group, err error) {
	resp, err := c.Do(http.MethodGet, groupPath, contentTypeJSON, nil)
	if err != nil {
		return
	}
	err = c.ReadJSON(resp, &groups)
	return
}

// GetGroup returns the group by the provided name
func (c *Client) GetGroup(name string) (group Group, err error) {
	if name == "" {
		err = errRequired("name")
		return
	}

	path := fmt.Sprintf("%s/%s", groupPath, name)
	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}

	err = c.ReadJSON(resp, &group)
	return
}

// CreateGroup creates a group
func (c *Client) CreateGroup(group *Group) error {
	if group.Name == "" {
		return errRequired("name")
	}
	if group.Namespaces == nil {
		group.Namespaces = make([]Namespace, 0)
	}

	payload, err := json.Marshal(group)
	if err != nil {
		return err
	}

	_, err = c.Do(http.MethodPost, groupPath, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return err
}

// DeleteGroup deletes a group
func (c *Client) DeleteGroup(name string) error {
	if name == "" {
		return errRequired("name")
	}

	path := fmt.Sprintf("%s/%s", groupPath, name)
	_, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)
	if err != nil {
		return err
	}
	return nil
}

// UpdateGroup updates a group
func (c *Client) UpdateGroup(group *Group) error {
	if group.Name == "" {
		return errRequired("name")
	}

	payload, err := json.Marshal(group)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/%s", groupPath, group.Name)
	_, err = c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return nil
}

// CloneGroup clones a group
func (c *Client) CloneGroup(currentName string, newName string) error {
	if currentName == "" {
		return errRequired("name")
	}
	if newName == "" {
		return errRequired("newName")
	}

	path := fmt.Sprintf("%s/%s/clone/%s", groupPath, currentName, newName)
	_, err := c.Do(http.MethodPost, path, contentTypeJSON, nil)
	if err != nil {
		return err
	}
	return nil
}
