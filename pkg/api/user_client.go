package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const usersPath = "api/v1/user"

// UserMember Lenses user
type UserMember struct {
	Username            string   `json:"username" yaml:"username" header:"Username"`
	Email               string   `json:"email,omitempty" yaml:"email" header:"Email"`
	Groups              []string `json:"groups" yaml:"groups" header:"Groups"`
	Password            string   `json:"password,omitempty" yaml:"password"`
	Type                string   `json:"type,omitempty" yaml:"security" header:"Security Type"`
	ManualGroupsMapping bool     `json:"manualGroupsMapping" yaml:"manualGroupsMapping" header:"Manual Group Mapping"`
}

type BasicUserMember struct {
	Username string   `json:"username" yaml:"username" header:"Username"`
	Email    string   `json:"email,omitempty" yaml:"email" header:"Email"`
	Groups   []string `json:"groups" yaml:"groups" header:"Groups"`
	Password string   `json:"password,omitempty" yaml:"password"`
	Type     string   `json:"type,omitempty" yaml:"security" header:"Security Type"`
}

type SSOLDAPUserMember struct {
	Username            string `json:"username" yaml:"username" header:"Username"`
	Email               string `json:"email,omitempty" yaml:"email" header:"Email"`
	Password            string `json:"password,omitempty" yaml:"password"`
	Type                string `json:"type,omitempty" yaml:"security" header:"Security Type"`
	ManualGroupsMapping bool   `json:"manualGroupsMapping" yaml:"manualGroupsMapping" header:"Manual Group Mapping"`
}

// GetUsers returns the list of users
func (c *Client) GetUsers() (users []UserMember, err error) {
	resp, err := c.Do(http.MethodGet, usersPath, contentTypeJSON, nil)
	if err != nil {
		return
	}
	err = c.ReadJSON(resp, &users)
	return
}

// GetUser returns the user by the provided name
func (c *Client) GetUser(name string) (user UserMember, err error) {
	if name == "" {
		err = errRequired("name")
		return
	}

	path := fmt.Sprintf("%s/%s", usersPath, name)
	resp, err := c.Do(http.MethodGet, path, contentTypeJSON, nil)
	if err != nil {
		return
	}

	err = c.ReadJSON(resp, &user)
	return
}

// CreateUser creates a user
func (c *Client) CreateUser(user *UserMember) error {
	if user.Username == "" {
		return errRequired("username")
	}
	// if len(user.Groups) == 0 {
	// 	return errRequired("groups")
	// }

	var payload []byte
	var err error

	if strings.ToLower(user.Type) == "basic" {
		bu := BasicUserMember{Username: user.Username, Password: user.Password, Type: user.Type, Email: user.Email, Groups: user.Groups}
		payload, err = json.Marshal(bu)
		if err != nil {
			return err
		}
	} else if (strings.ToLower(user.Type) == "sso" || strings.ToLower(user.Type) == "ldap") && !user.ManualGroupsMapping {
		bu := SSOLDAPUserMember{Username: user.Username, Password: user.Password, Type: user.Type, Email: user.Email, ManualGroupsMapping: user.ManualGroupsMapping}
		payload, err = json.Marshal(bu)
		if err != nil {
			return err
		}
	} else {
		payload, err = json.Marshal(user)
		if err != nil {
			return err
		}
	}

	_, err = c.Do(http.MethodPost, usersPath, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return err
}

// DeleteUser deletes a user
func (c *Client) DeleteUser(username string) error {
	if username == "" {
		return errRequired("name")
	}

	path := fmt.Sprintf("%s/%s", usersPath, username)
	_, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)
	if err != nil {
		return err
	}
	return nil
}

// UpdateUser updates a user
func (c *Client) UpdateUser(user *UserMember) error {
	if user.Username == "" {
		return errRequired("name")
	}

	var payload []byte
	var err error
	if strings.ToLower(user.Type) == "basic" {
		bu := BasicUserMember{Username: user.Username, Password: user.Password, Type: user.Type, Email: user.Email, Groups: user.Groups}
		payload, err = json.Marshal(bu)
		if err != nil {
			return err
		}
	} else if (strings.ToLower(user.Type) == "sso" || strings.ToLower(user.Type) == "ldap") && !user.ManualGroupsMapping {
		bu := SSOLDAPUserMember{Username: user.Username, Password: user.Password, Type: user.Type, Email: user.Email, ManualGroupsMapping: user.ManualGroupsMapping}
		payload, err = json.Marshal(bu)
		if err != nil {
			return err
		}
	} else {
		payload, err = json.Marshal(user)
		if err != nil {
			return err
		}
	}

	path := fmt.Sprintf("%s/%s", usersPath, user.Username)
	_, err = c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return nil
}

type changePassword struct {
	Value string `json:"value"`
}

// UpdateUserPassword updaes the password of a user
func (c *Client) UpdateUserPassword(username, password string) error {
	if username == "" {
		return errRequired("name")
	}
	if password == "" {
		return errRequired("password")
	}

	payload, err := json.Marshal(changePassword{Value: password})
	if err != nil {
		return err
	}

	path := fmt.Sprintf("%s/%s/password", usersPath, username)
	_, err = c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}
	return nil
}
