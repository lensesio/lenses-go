package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const serviceAccountPath = "api/v1/serviceaccount"

//ServiceAccount the service account data transfer object
type ServiceAccount struct {
	Name   string   `json:"name" yaml:"name" header:"Name"`
	Owner  string   `json:"owner" yaml:"owner" header:"Owner"`
	Groups []string `json:"groups" yaml:"groups" header:"Groups"`
}

//GetServiceAccounts returns the list of service accounts
func (c *Client) GetServiceAccounts() (serviceAccounts []ServiceAccount, err error) {
	resp, err := c.Do(http.MethodGet, serviceAccountPath, contentTypeJSON, nil)
	if err != nil {
		return
	}
	err = c.ReadJSON(resp, &serviceAccounts)
	return
}

//GetServiceAccount returns the service account by the provided name
func (c *Client) GetServiceAccount(name string) (serviceAccount ServiceAccount, err error) {
	if name == "" {
		err = errRequired("name")
		return
	}

	path := fmt.Sprintf("%s/%s", serviceAccountPath, name)
	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}

	err = c.ReadJSON(resp, &serviceAccount)
	return
}

//CreateServiceAccount creates a service account
func (c *Client) CreateServiceAccount(serviceAccount *ServiceAccount) error {
	if serviceAccount.Name == "" {
		return errRequired("name")
	}
	if len(serviceAccount.Groups) == 0 {
		return errRequired("groups")
	}

	payload, err := json.Marshal(serviceAccount)
	if err != nil {
		return err
	}

	_, err = c.Do(http.MethodPost, serviceAccountPath, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return err
}

//DeleteServiceAccount deletes a service account
func (c *Client) DeleteServiceAccount(name string) error {
	if name == "" {
		return errRequired("name")
	}

	path := fmt.Sprintf("%s/%s", serviceAccountPath, name)
	_, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)
	if err != nil {
		return err
	}
	return nil
}

//UpdateServiceAccount updates a service account
func (c *Client) UpdateServiceAccount(serviceAccount *ServiceAccount) error {
	if serviceAccount.Name == "" {
		return errRequired("name")
	}
	if len(serviceAccount.Groups) == 0 {
		return errRequired("groups")
	}

	payload, err := json.Marshal(serviceAccount)
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/%s", serviceAccountPath, serviceAccount.Name)
	_, err = c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return nil
}

//GetServiceAccountToken returns the service account token for the provided name
func (c *Client) GetServiceAccountToken(name string) (token string, err error) {
	if name == "" {
		err = errRequired("name")
		return
	}
	path := fmt.Sprintf("%s/%s/token", serviceAccountPath, name)
	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}

	tokenBytes, err := c.ReadResponseBody(resp)
	if err != nil {
		return
	}
	token = string(tokenBytes)
	return
}

//RevokeServiceAccountToken returns the service account token for the provided name
func (c *Client) RevokeServiceAccountToken(name string) error {
	if name == "" {
		return errRequired("name")
	}
	path := fmt.Sprintf("%s/%s/revoke", serviceAccountPath, name)
	_, err := c.Do(http.MethodPut, path, contentTypeJSON, nil)
	if err != nil {
		return err
	}
	return nil
}
