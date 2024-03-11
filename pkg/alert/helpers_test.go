package alert

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/stretchr/testify/assert"
)

func TestAlertSettingsV1(t *testing.T) {
	// Given a Lenses instance that has the v1 endpoint,
	m := http.NewServeMux()
	m.HandleFunc("/api/v1/alert/settings", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(getV1AlertSettingsResponse))
	})
	s := httptest.NewServer(m)
	cl, err := api.OpenConnection(api.ClientConfig{
		Host:  s.URL,
		Token: "ignored", // Ignored.
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("GetConsumer", func(t *testing.T) {
		// When the consumer alert settings are requested,
		con, err := GetConsumerAlertSettings(cl)
		if err != nil {
			t.Fatal(err)
		}
		// Then they look as expected.
		assert.Equal(t, api.ConsumerAlertSettings{
			ID:          2000,
			Description: "Consumer Lag exceeded",
			ConditionDetails: []api.ConsumerAlertConditionRequestv1{
				{
					Condition: api.ConsumerConditionDsl{
						Group:     "bananas-v1-group",
						Threshold: 7,
						Topic:     "bananas-v1-topic",
						Mode:      api.PerPartitionMode,
					},
					Channels: []string{"bananas-v1-channel"},
				},
			},
		}, con)
	})
	t.Run("GetProducer", func(t *testing.T) {
		// When the producer alert settings are requested,
		con, err := GetProducerAlertSettings(cl)
		if err != nil {
			t.Fatal(err)
		}
		// Then they look as expected.
		assert.Equal(t, api.ProducerAlertSettings{
			ID:          5000,
			Description: "Data Produced",
			ConditionDetails: []api.AlertConditionRequestv1{
				{
					Condition: api.DataProduced{
						ConnectionName: "kafka",
						DatasetName:    "bananas-v1-topic",
						Threshold: api.Threshold{
							Type:     "more_than",
							Messages: 12,
						},
						Duration: "PT10M",
					},
				},
			},
		}, con)
	})
}

func TestAlertSettingsV2(t *testing.T) {
	// Given a Lenses instance that only has the v2 endpoint,
	m := http.NewServeMux()
	m.HandleFunc("/api/v2/alert/settings", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(getV2AlertSettingsResponse))
	})
	s := httptest.NewServer(m)
	defer s.Close()
	cl, err := api.OpenConnection(api.ClientConfig{
		Host:  s.URL,
		Token: "yolo",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Run("GetConsumer", func(t *testing.T) {
		// When the consumer alert settings are requested,
		con, err := GetConsumerAlertSettings(cl)
		if err != nil {
			t.Fatal(err)
		}
		// Then they look as expected.
		assert.Equal(t, api.ConsumerAlertSettings{
			ID:          2000,
			Description: "Consumer Lag exceeded",
			ConditionDetails: []api.ConsumerAlertConditionRequestv1{
				{
					Condition: api.ConsumerConditionDsl{
						Group:     "bananas-v2-group",
						Threshold: 1000,
						Topic:     "bananas-v2-topic",
					},
					Channels: []string{"bananas-v2-channel-1", "bananas-v2-channel-2"},
				},
			},
		}, con)
	})
	t.Run("GetProducer", func(t *testing.T) {
		// When the producer alert settings are requested,
		con, err := GetProducerAlertSettings(cl)
		if err != nil {
			t.Fatal(err)
		}
		// Then they look as expected.
		assert.Equal(t, api.ProducerAlertSettings{
			ID:          5000,
			Description: "Data Produced",
			ConditionDetails: []api.AlertConditionRequestv1{
				{
					Condition: api.DataProduced{
						ConnectionName: "kafka",
						DatasetName:    "bananas-v2-topic",
						Threshold: api.Threshold{
							Type:     "more_than",
							Messages: 8,
						},
						Duration: "PT5M",
					},
				},
			},
		}, con)
	})
}

// getV1AlertSettingsResponse is an actual response grabbed from Lenses.
const getV1AlertSettingsResponse = `
{
    "categories": {
        "Kafka Connect": [
            {
                "id": 3000,
                "description": "Connector deleted",
                "category": "Kafka Connect",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            }
        ],
        "Infrastructure": [
            {
                "id": 1000,
                "description": "Kafka Broker is down",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1001,
                "description": "Zookeeper Node is down",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1002,
                "description": "Connect Worker is down",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1003,
                "description": "Schema Registry is down",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1005,
                "description": "Under replicated partitions",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1006,
                "description": "Partitions offline",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1007,
                "description": "Active Controllers",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1008,
                "description": "Multiple Broker Versions",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1009,
                "description": "File-open descriptors high capacity on Brokers",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1010,
                "description": "Average % the request handler is idle",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1011,
                "description": "Fetch requests failure",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1012,
                "description": "Produce requests failure",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1013,
                "description": "Broker disk usage is greater than the cluster average",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 1014,
                "description": "Leader Imbalance",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            }
        ],
        "Topics": [
            {
                "id": 4000,
                "description": "Topic has been created",
                "category": "Topics",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 4001,
                "description": "Topic has been deleted",
                "category": "Topics",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            },
            {
                "id": 4002,
                "description": "Topic data has been deleted.",
                "category": "Topics",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            }
        ],
        "Consumers": [
            {
                "id": 2000,
                "description": "Consumer Lag exceeded",
                "category": "Consumers",
                "enabled": true,
                "conditionTemplate": "lag >= $Threshold-Number on group $Consumer-Group and topic $Topic-Name",
                "conditionRegex": "lag >= ([1-9][0-9]*) on group (\\b\\S+\\b) and topic (\\b\\S+\\b)",
                "docs": "Raises an alert when the consumer lag exceeds the threshold on any partition.",
                "conditions": {
                    "e7cef036-029f-4954-8c98-96be01a5a86d": "lag >= 7 on group bananas-v1-group and topic bananas-v1-topic"
                },
                "channels": [],
                "conditionDetails": {
                    "e7cef036-029f-4954-8c98-96be01a5a86d": {
                        "createdAt": "2024-03-07T10:21:39.439391Z",
                        "createdBy": "admin",
                        "modifiedAt": "2024-03-07T10:21:39.439391Z",
                        "modifiedBy": "admin",
                        "channels": [
                            {
                                "id": "287e9625-1767-43cf-8975-b2c0754e3ddd",
                                "name": "bananas-v1-channel",
                                "templateName": "Webhook"
                            }
                        ],
                        "conditionDsl": {
                            "group": "bananas-v1-group",
                            "topic": "bananas-v1-topic",
                            "threshold": 7,
                            "mode": "PerPartitionMode"
                        },
                        "conditionState": null
                    }
                },
                "isAvailable": true
            }
        ],
        "Data Produced": [
            {
                "id": 5000,
                "description": "Data Produced",
                "category": "Data Produced",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": "Raises an alert when the data produced on a topic doesn't match expected threshold",
                "conditions": {},
                "channels": [],
				"conditionDetails": {
                    "3d8b3511-716a-4703-9c34-d6b41c29f561": {
                        "createdAt": "2024-03-05T14:14:44.474717Z",
                        "createdBy": "admin",
                        "modifiedAt": "2024-03-05T14:14:44.474717Z",
                        "modifiedBy": "admin",
                        "channels": [],
                        "conditionDsl": {
                            "connectionName": "kafka",
                            "datasetName": "bananas-v1-topic",
                            "threshold": {
                                "type": "more_than",
                                "messages": 12
                            },
                            "duration": "PT10M"
                        },
                        "conditionState": null
                    }
                },
                "isAvailable": true
            }
        ],
        "Apps": [
            {
                "id": 6000,
                "description": "Connector Failed",
                "category": "Apps",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "conditions": null,
                "channels": [],
                "conditionDetails": {},
                "isAvailable": true
            }
        ]
    }
}`

// getV2AlertSettingsResponse is an actual response grabbed from Lenses.
const getV2AlertSettingsResponse = `
{
    "categories": {
        "Kafka Connect": [
            {
                "id": 3000,
                "description": "Connector deleted",
                "category": "Kafka Connect",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.783498Z"
                }
            }
        ],
        "Infrastructure": [
            {
                "id": 1000,
                "description": "Kafka Broker is down",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.771723Z"
                }
            },
            {
                "id": 1001,
                "description": "Zookeeper Node is down",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.772693Z"
                }
            },
            {
                "id": 1002,
                "description": "Connect Worker is down",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.773614Z"
                }
            },
            {
                "id": 1003,
                "description": "Schema Registry is down",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.774614Z"
                }
            },
            {
                "id": 1005,
                "description": "Under replicated partitions",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.775783Z"
                }
            },
            {
                "id": 1006,
                "description": "Partitions offline",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.776646Z"
                }
            },
            {
                "id": 1007,
                "description": "Active Controllers",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.777460Z"
                }
            },
            {
                "id": 1008,
                "description": "Multiple Broker Versions",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.778287Z"
                }
            },
            {
                "id": 1009,
                "description": "File-open descriptors high capacity on Brokers",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.779086Z"
                }
            },
            {
                "id": 1010,
                "description": "Average % the request handler is idle",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.779891Z"
                }
            },
            {
                "id": 1011,
                "description": "Fetch requests failure",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.780537Z"
                }
            },
            {
                "id": 1012,
                "description": "Produce requests failure",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.781133Z"
                }
            },
            {
                "id": 1013,
                "description": "Broker disk usage is greater than the cluster average",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.781757Z"
                }
            },
            {
                "id": 1014,
                "description": "Leader Imbalance",
                "category": "Infrastructure",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.782366Z"
                }
            }
        ],
        "Topics": [
            {
                "id": 4000,
                "description": "Topic has been created",
                "category": "Topics",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.784084Z"
                }
            },
            {
                "id": 4001,
                "description": "Topic has been deleted",
                "category": "Topics",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.784758Z"
                }
            },
            {
                "id": 4002,
                "description": "Topic data has been deleted.",
                "category": "Topics",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": null,
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": null,
                    "conditionDsl": null,
                    "modifiedBy": "Lenses",
                    "modifiedAt": "2024-03-12T13:35:23.785343Z"
                }
            }
        ],
        "Consumers": [
            {
                "id": 2000,
                "description": "Consumer Lag exceeded",
                "category": "Consumers",
                "enabled": true,
                "conditionTemplate": "lag >= $Threshold-Number on group $Consumer-Group and topic $Topic-Name",
                "conditionRegex": "lag >= ([1-9][0-9]*) on group (\\b\\S+\\b) and topic (\\b\\S+\\b)",
                "docs": "Raises an alert when the consumer lag exceeds the threshold on any partition.",
                "isAvailable": true,
                "details": {
                    "conditions": {
                        "750148a9-88d3-402a-ae26-256cc0fc4c37": {
                            "createdAt": "2020-10-07T15:24:19.331Z",
                            "createdBy": "admin",
                            "modifiedAt": "2020-12-10T17:10:30.488Z",
                            "modifiedBy": "admin",
                            "channels": [
                                {
                                    "id": "3187a395-a73b-42ec-b5d0-acf0176470a1",
                                    "name": "bananas-v2-channel-1",
                                    "templateName": "Webhook"
                                },
                                {
                                    "id": "7c627b1e-dacc-4aa0-835f-43733c9dc783",
                                    "name": "bananas-v2-channel-2",
                                    "templateName": "Slack"
                                }
                            ],
                            "condition": "lag >= 1000 on group bananas-v2-group and topic bananas-v2-topic",
                            "conditionDsl": {
                                "group": "bananas-v2-group",
                                "topic": "bananas-v2-topic",
                                "threshold": 1000
                            },
                            "conditionState": null
                        }
                    },
                    "alertType": "Conditional"
                }
            }
        ],
        "Data Produced": [
            {
                "id": 5000,
                "description": "Data Produced",
                "category": "Data Produced",
                "enabled": true,
                "conditionTemplate": null,
                "conditionRegex": null,
                "docs": "Raises an alert when the data produced on a topic doesn't match expected threshold",
                "isAvailable": true,
                "details": {
                    "conditions": {
                        "8456a682-d86d-4761-b5c6-a8bbe703699d": {
                            "createdAt": "2024-03-05T13:45:16.443486Z",
                            "createdBy": "admin",
                            "modifiedAt": "2024-03-05T13:45:16.443486Z",
                            "modifiedBy": "admin",
                            "channels": [],
                            "condition": "<error>",
                            "conditionDsl": {
                                "connectionName": "kafka",
                                "datasetName": "bananas-v2-topic",
                                "threshold": {
                                    "type": "more_than",
                                    "messages": 8
                                },
                                "duration": "PT5M"
                            },
                            "conditionState": null
                        }
                    },
                    "alertType": "Conditional"
                }
            }
        ],
        "Apps": [
            {
                "id": 6000,
                "description": "Connector Failed",
                "category": "Apps",
                "enabled": true,
                "conditionTemplate": "enabled|disabled. after >= $Max-Restarts restarts each with a grace period of $Grace-Period seconds",
                "conditionRegex": "(enabled|disabled). after >= ([1-9][0-9]*) restarts each with a grace period of ([1-9][0-9]*) seconds",
                "docs": "Raises the alert if the connector has tasks in FAILED state after N numbers of attempts to restart it.",
                "isAvailable": true,
                "details": {
                    "alertType": "Fixed",
                    "channels": [],
                    "condition": "disabled. after >= 0 restarts each with a grace period of 180 seconds.",
                    "conditionDsl": {
                        "enabled": false,
                        "restarts": 0,
                        "gracePeriod": 180
                    },
                    "modifiedBy": "lenses-system",
                    "modifiedAt": "2024-02-27T14:58:41.958784Z"
                }
            }
        ]
    }
}`
