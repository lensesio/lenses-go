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
	Owner  string   `json:"owner,omitempty" yaml:"owner,omitempty" header:"Owner"`
	Groups []string `json:"groups" yaml:"groups" header:"Groups"`
}

//CreateSvcAccPayload the data transfer object when we create a new service account
type CreateSvcAccPayload struct {
	Token string `json:"token,omitempty"`
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
func (c *Client) CreateServiceAccount(serviceAccount *ServiceAccount) (token CreateSvcAccPayload, err error) {
	if serviceAccount.Name == "" {
		err = errRequired("name")
		return
	}
	if len(serviceAccount.Groups) == 0 {
		err = errRequired("groups")
		return
	}

	payload, err := json.Marshal(serviceAccount)
	if err != nil {
		return
	}

	resp, err := c.Do(http.MethodPost, serviceAccountPath, contentTypeJSON, payload)
	if err != nil {
		return
	}
	err = c.ReadJSON(resp, &token)
	return
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

//RevokeServiceAccountToken returns the service account token for the provided name
func (c *Client) RevokeServiceAccountToken(name string, newToken string) (token CreateSvcAccPayload, err error) {
	if name == "" {
		err = errRequired("name")
		return
	}
	payload, err := json.Marshal(CreateSvcAccPayload{Token: newToken})
	if err != nil {
		return
	}

	path := fmt.Sprintf("%s/%s/revoke", serviceAccountPath, name)
	resp, err := c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return
	}
	err = c.ReadJSON(resp, &token)
	return
}
