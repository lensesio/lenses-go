package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/lensesio/lenses-go/pkg"
)

// LicenseInfo describes the data received from the `GetLicenseInfo`.
type LicenseInfo struct {
	ClientID    string `json:"clientId" header:"ID,text"`
	IsRespected bool   `json:"isRespected" header:"Respected"`
	MaxBrokers  int    `json:"maxBrokers" header:"Max Brokers"`
	MaxMessages int    `json:"maxMessages,omitempty" header:"/ Messages"`
	Expiry      int64  `json:"expiry" header:"Expires,timestamp(ms|02 Jan 2006 15:04)"`

	// no-payload data.

	// ExpiresAt is the time.Time expiration datetime (unix).
	ExpiresAt time.Time `json:"-"`

	// ExpiresDur is the duration that expires from now.
	ExpiresDur time.Duration `json:"-"`

	// YearsToExpire is the length of years that expires from now.
	YearsToExpire int `json:"yearsToExpire,omitempty"`
	// MonthsToExpire is the length of months that expires from now.
	MonthsToExpire int `json:"monthsToExpire,omitempty"`
	// DaysToExpire is the length of days that expires from now.
	DaysToExpire int `json:"daysToExpire,omitempty"`
}

// License is the JSON payload for updating a license.
type License struct {
	Source   string `json:"source"`
	ClientID string `json:"clientId"`
	Details  string `json:"details"`
	Key      string `json:"key"`
}

// GetLicenseInfo returns the license information for the connected lenses box.
func (c *Client) GetLicenseInfo() (LicenseInfo, error) {
	var lc LicenseInfo

	resp, err := c.Do(http.MethodGet, pkg.LicensePath, "", nil)
	if err != nil {
		return lc, err
	}

	if err = c.ReadJSON(resp, &lc); err != nil {
		return lc, err
	}

	lc.ExpiresAt = time.Unix(lc.Expiry/1000, 0)
	lc.ExpiresDur = lc.ExpiresAt.Sub(time.Now())
	lc.DaysToExpire = int(lc.ExpiresDur.Hours() / 24)
	lc.MonthsToExpire = int(lc.DaysToExpire / 30)
	lc.YearsToExpire = int(lc.MonthsToExpire / 12)

	if lc.YearsToExpire > 0 {
		lc.DaysToExpire = 0
		lc.MonthsToExpire = 0
	} else if lc.MonthsToExpire > 0 {
		lc.DaysToExpire = 0
	}

	return lc, nil
}

// UpdateLicense handles the `PUT` API call to update a license at runtime
func (c *Client) UpdateLicense(license License) error {

	payload, err := json.Marshal(license)
	resp, err := c.Do(http.MethodPut, pkg.LicensePath, contentTypeJSON, payload)
	if err != nil {
		return err

	}
	return resp.Body.Close()
}
