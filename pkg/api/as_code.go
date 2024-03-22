package api

import (
	"encoding/json"
	"fmt"
	"net/http"
)

const resourcePath = "api/v1/resource"
const resourceKafkaPath = resourcePath + "/kafka"
const resourceConnectBasePath = resourceKafkaPath + "/connect" //api/v1/resource/kafka/connect/{connect-cluster-name}/connector/{connector-name}"

// Returns the connector as code
func (c *Client) GetConnectorAsCode(cluster string, connector string) (yaml string, err error) {
	path := fmt.Sprintf("%s/%s/connector/%s", resourceConnectBasePath, cluster, connector)
	resp, err := c.Do(http.MethodGet, path, contentTypeYaml, nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		var errorResponse ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return "", fmt.Errorf("failed to decode error response: %w", err)
		}
		return "", fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, errorResponse.Error)
	}

	bodyBytes, err := c.ReadResponseBody(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	yaml = string(bodyBytes)

	return yaml, nil
}

func (c *Client) ImportResource(yaml string) error {
	resp, err := c.Do(http.MethodPut, resourcePath, contentTypeYaml, []byte(yaml))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusCreated:
		return nil
	case http.StatusBadRequest, http.StatusUnauthorized, http.StatusPaymentRequired, http.StatusForbidden, http.StatusNotFound, http.StatusInternalServerError:
		var errorResponse ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errorResponse); err != nil {
			return fmt.Errorf("failed to decode error response: %w", err)
		}
		return fmt.Errorf("request failed with status code %d: %s", resp.StatusCode, errorResponse.Error)
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}
