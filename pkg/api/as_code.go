package api

import (
	"fmt"
	"net/http"
)

const resourcePath = "api/v1/resource"
const resourceKafkaPath = resourcePath + "/kafka"
const resourceConnectBasePath = resourceKafkaPath + "/connect" //api/v1/resource/kafka/connect/{connect-cluster-name}/connector/{connector-name}"

// Returns the connector as code
func (c *Client) GetConnectorAsCode(cluster string, connector string) (yaml string, err error) {
	path := fmt.Sprintf("%s/%s/connector/%s", resourceConnectBasePath, cluster, connector)
	//c.Do handles the error handling
	resp, err := c.Do(http.MethodGet, path, contentTypeYaml, nil)
	if err != nil {
		return "", err
	}

	defer resp.Body.Close()

	bodyBytes, err := c.ReadResponseBody(resp)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	yaml = string(bodyBytes)

	return yaml, nil
}

func (c *Client) ImportResource(yaml string) error {
	// c.Do handles the error handling
	_, err := c.Do(http.MethodPut, resourcePath, contentTypeYaml, []byte(yaml))
	if err != nil {
		return err
	}
	return nil
}
