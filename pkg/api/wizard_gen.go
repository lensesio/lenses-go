package api

import "net/http"

type SetupStatus struct {
	IsCompleted bool `json:"isCompleted"` // Required.
	IsLicensed  bool `json:"isLicensed"`  // Required.
}

// Returns the setup stage status.
// Tags: Internal.
func (c *Client) GetSetupStatus() (resp SetupStatus, err error) {
	err = c.do(
		http.MethodGet,
		"/api/v1/setup",
		nil,   // request
		&resp, // response
	)
	return
}
