package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/lensesio/lenses-go/v5/pkg"
)

// SingleTopicOffset represent the payload structure
// of the API for updating a single partition of a single topic.
type SingleTopicOffset struct {
	Type   string `json:"type" yaml:"type"`
	Offset int    `json:"offset,omitempty" yaml:"offset"`
}

// MultipleTopicOffsets represent the payload structure
// of the API for updating all partitions of multiple topics.
type MultipleTopicOffsets struct {
	Type   string   `json:"type" yaml:"type"`
	Target string   `json:"target,omitempty" yaml:"type"`
	Topics []string `json:"topics,omitempty" yaml:"topics"`
}

// UpdateSingleTopicOffset handles the API call to update
// a signle partition of a topic.
func (c *Client) UpdateSingleTopicOffset(groupID, topic, partitionID, offsetType string, offset int) error {
	if offsetType == "" {
		return errRequired("field `type` is missing")
	}

	path := fmt.Sprintf("%s/%s/offsets/topics/%s/partitions/%s", pkg.ConsumersGroupPath, groupID, topic, partitionID)
	singleTopic := SingleTopicOffset{Type: offsetType, Offset: offset}
	payload, err := json.Marshal(singleTopic)

	_, err = c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}

	return nil
}

// UpdateMultipleTopicsOffset handles the Lenses API call to update
// all partitions of multiple topics of a consumer group.
func (c *Client) UpdateMultipleTopicsOffset(groupID, offsetType, target string, topics []string) error {
	path := fmt.Sprintf("%s/%s/offsets", pkg.ConsumersGroupPath, groupID)
	multipleTopics := MultipleTopicOffsets{Type: offsetType, Target: target, Topics: topics}
	payload, err := json.Marshal(multipleTopics)

	_, err = c.Do(http.MethodPut, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}

	return nil
}

// DeleteSingleTopicOffset handles the API call to delete consumer group offsets for
// a signle partition of a topic.
func (c *Client) DeleteSingleTopicOffset(groupID, topic, partitionID string) error {

	path := fmt.Sprintf("%s/%s/topics/%s/partitions/%s/offsets", pkg.ConsumersGroupPath, groupID, topic, partitionID)

	_, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)
	if err != nil {
		return err
	}

	return nil
}

// DeleteMultipleTopicsOffset handles the Lenses API call to delete
// all partitions of multiple topics of a consumer group.
func (c *Client) DeleteMultipleTopicsOffset(groupID string, topics []string) error {
	path := fmt.Sprintf("%s/%s/offsets/delete", pkg.ConsumersGroupPath, groupID)
	multipleTopics := MultipleTopicOffsets{Topics: topics}
	payload, err := json.Marshal(multipleTopics)

	_, err = c.Do(http.MethodPost, path, contentTypeJSON, payload)
	if err != nil {
		return err
	}

	return nil
}

// DeleteConsumerGroup handles the Lenses API call to delete a specific consumer group
func (c *Client) DeleteConsumerGroup(groupID string) error {
	path := fmt.Sprintf("%s/%s", pkg.ConsumersGroupPath, groupID)

	_, err := c.Do(http.MethodDelete, path, contentTypeJSON, nil)
	if err != nil {
		return err
	}

	return nil
}
