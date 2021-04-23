package api

import (
	"net/http"

	"fmt"
	"net/url"
	"strconv"
)

// As Alert and Audit channels have the same API contracts,
// some parts of the code might be reused.

func (c *Client) GetChannels(path string, page int, pageSize int, sortField, sortOrder, templateName, channelName string) (response AlertChannelResponse, err error) {
	queryString := constructQueryString(path, page, pageSize, sortField, sortOrder, templateName, channelName)
	resp, err := c.Do(http.MethodGet, queryString, contentTypeJSON, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.ReadJSON(resp, &response)
	return
}

func (c *Client) GetChannelsWithDetails(path string, page int, pageSize int, sortField, sortOrder, templateName, channelName string) (response AlertChannelResponseWithDetails, err error) {
	queryString := constructQueryString(path, page, pageSize, sortField, sortOrder, templateName, channelName)
	resp, err := c.Do(http.MethodGet, queryString, contentTypeJSON, nil)
	if err != nil {
		return
	}
	defer resp.Body.Close()
	err = c.ReadJSON(resp, &response)
	return
}

// TODO make it private
func constructQueryString(path string, page int, pageSize int, sortField, sortOrder, templateName, channelName string) (query string) {
	v := url.Values{}
	v.Add("pageSize", strconv.Itoa(pageSize))

	if page != 0 {
		v.Add("page", strconv.Itoa(page))
	}
	if sortField != "" {
		v.Add("sortField", sortField)
	}
	if sortOrder != "" {
		v.Add("sortOrder", sortOrder)
	}
	if templateName != "" {
		v.Add("templateName", templateName)
	}
	if channelName != "" {
		v.Add("channelName", channelName)
	}

	query = fmt.Sprintf("%s?%s", path, v.Encode())
	return
}
