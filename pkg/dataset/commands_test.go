package dataset

import (
	"net/http"
	"testing"

	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/test"
	"github.com/stretchr/testify/assert"
)

const datasetResponse = `
{
  "applications": [],
  "config": [
    {
      "defaultValue": "more than 1000y",
      "documentation": null,
      "isDefault": true,
      "name": "message.timestamp.difference.max.ms",
      "originalValue": "9223372036854775807",
      "value": "more than 1000y"
    },
    {
      "defaultValue": "1 MB",
      "documentation": null,
      "isDefault": true,
      "name": "max.message.bytes",
      "originalValue": "1000012",
      "value": "1 MB"
    },
    {
      "defaultValue": "10.5 MB",
      "documentation": null,
      "isDefault": true,
      "name": "segment.index.bytes",
      "originalValue": "10485760",
      "value": "10.5 MB"
    },
    {
      "defaultValue": "0ms",
      "documentation": null,
      "isDefault": true,
      "name": "segment.jitter.ms",
      "originalValue": "0",
      "value": "0ms"
    },
    {
      "defaultValue": "0.5",
      "documentation": null,
      "isDefault": true,
      "name": "min.cleanable.dirty.ratio",
      "originalValue": "0.5",
      "value": "0.5"
    },
    {
      "defaultValue": "-1 Bytes",
      "documentation": null,
      "isDefault": false,
      "name": "retention.bytes",
      "originalValue": "26214400",
      "value": "26.2 MB"
    },
    {
      "defaultValue": "",
      "documentation": null,
      "isDefault": true,
      "name": "follower.replication.throttled.replicas",
      "originalValue": "",
      "value": ""
    },
    {
      "defaultValue": "1m",
      "documentation": null,
      "isDefault": true,
      "name": "file.delete.delay.ms",
      "originalValue": "60000",
      "value": "1m"
    },
    {
      "defaultValue": null,
      "documentation": null,
      "isDefault": true,
      "name": "message.downconversion.enable",
      "originalValue": "true",
      "value": "true"
    },
    {
      "defaultValue": "producer",
      "documentation": null,
      "isDefault": false,
      "name": "compression.type",
      "originalValue": "gzip",
      "value": "gzip"
    },
    {
      "defaultValue": "0ms",
      "documentation": null,
      "isDefault": true,
      "name": "min.compaction.lag.ms",
      "originalValue": "0",
      "value": "0ms"
    },
    {
      "defaultValue": "more than 1000y",
      "documentation": null,
      "isDefault": true,
      "name": "flush.ms",
      "originalValue": "9223372036854775807",
      "value": "more than 1000y"
    },
    {
      "defaultValue": "delete",
      "documentation": null,
      "isDefault": true,
      "name": "cleanup.policy",
      "originalValue": "delete",
      "value": "delete"
    },
    {
      "defaultValue": "CreateTime",
      "documentation": null,
      "isDefault": true,
      "name": "message.timestamp.type",
      "originalValue": "CreateTime",
      "value": "CreateTime"
    },
    {
      "defaultValue": "true",
      "documentation": null,
      "isDefault": true,
      "name": "unclean.leader.election.enable",
      "originalValue": "false",
      "value": "false"
    },
    {
      "defaultValue": "9223372036854775807",
      "documentation": null,
      "isDefault": true,
      "name": "flush.messages",
      "originalValue": "9223372036854775807",
      "value": "9223372036854775807"
    },
    {
      "defaultValue": "7d",
      "documentation": null,
      "isDefault": true,
      "name": "retention.ms",
      "originalValue": "604800000",
      "value": "7d"
    },
    {
      "defaultValue": "1",
      "documentation": null,
      "isDefault": true,
      "name": "min.insync.replicas",
      "originalValue": "1",
      "value": "1"
    },
    {
      "defaultValue": "Relative to broker version",
      "documentation": null,
      "isDefault": true,
      "name": "message.format.version",
      "originalValue": "2.0-IV1",
      "value": "2.0-IV1"
    },
    {
      "defaultValue": "",
      "documentation": null,
      "isDefault": true,
      "name": "leader.replication.throttled.replicas",
      "originalValue": "",
      "value": ""
    },
    {
      "defaultValue": "1d",
      "documentation": null,
      "isDefault": true,
      "name": "delete.retention.ms",
      "originalValue": "86400000",
      "value": "1d"
    },
    {
      "defaultValue": "false",
      "documentation": null,
      "isDefault": true,
      "name": "preallocate",
      "originalValue": "false",
      "value": "false"
    },
    {
      "defaultValue": "4.1 KB",
      "documentation": null,
      "isDefault": true,
      "name": "index.interval.bytes",
      "originalValue": "4096",
      "value": "4.1 KB"
    },
    {
      "defaultValue": "1.1 GB",
      "documentation": null,
      "isDefault": false,
      "name": "segment.bytes",
      "originalValue": "8388608",
      "value": "8.4 MB"
    },
    {
      "defaultValue": "7d",
      "documentation": null,
      "isDefault": true,
      "name": "segment.ms",
      "originalValue": "604800000",
      "value": "7d"
    }
  ],
  "consumers": [],
  "isCompacted": false,
  "isControlTopic": false,
  "isMarkedForDeletion": false,
  "keySchema": null,
  "keySchemaInlined": null,
  "keySchemaVersion": null,
  "keyType": "BYTES",
  "messagesPerPartition": [
    {
      "begin": 0,
      "end": 13515,
      "messages": 13515,
      "partition": 0
    },
    {
      "begin": 0,
      "end": 13351,
      "messages": 13351,
      "partition": 1
    },
    {
      "begin": 0,
      "end": 13480,
      "messages": 13480,
      "partition": 2
    },
    {
      "begin": 0,
      "end": 13889,
      "messages": 13889,
      "partition": 3
    },
    {
      "begin": 0,
      "end": 13452,
      "messages": 13452,
      "partition": 4
    }
  ],
  "messagesPerSecond": 21,
  "partitions": 5,
  "permissions": [
    "ShowTopic",
    "CreateTopic",
    "RequestTopicCreation",
    "DropTopic",
    "ConfigureTopic",
    "QueryTopic",
    "InsertData",
    "DeleteData",
    "UpdateSchema",
    "ViewSchema",
    "UpdateMetadata"
  ],
  "replication": 1,
  "description": null,
  "timestamp": 1605564013913,
  "topicName": "nyc_yellow_taxi_trip_data",
  "totalMessages": 67687,
  "valueSchema": "{\"type\":\"record\",\"name\":\"trip_record\",\"namespace\":\"com.landoop.transportation.nyc.trip.yellow\",\"doc\":\"Schema for yellow taxi trip records from NYC TLC data. [http://www.nyc.gov/html/tlc/html/about/trip_record_data.shtml]\",\"fields\":[{\"name\":\"VendorID\",\"type\":\"int\",\"doc\":\"A code indicating the TPEP provider that provided the record. 1: Creative Mobile Technologies, LLC 2: VeriFone Inc.\"},{\"name\":\"tpep_pickup_datetime\",\"type\":\"string\",\"doc\":\"The date and time when the meter was engaged.\"},{\"name\":\"tpep_dropoff_datetime\",\"type\":\"string\",\"doc\":\"The date and time when the meter was disengaged.\"},{\"name\":\"passenger_count\",\"type\":\"int\",\"doc\":\"The number of passengers in the vehicle. This is a driver-entered value.\"},{\"name\":\"trip_distance\",\"type\":\"double\",\"doc\":\"The elapsed trip distance in miles reported by the taximeter.\"},{\"name\":\"pickup_longitude\",\"type\":\"double\",\"doc\":\"Longitude where the meter was engaged.\"},{\"name\":\"pickup_latitude\",\"type\":\"double\",\"doc\":\"Latitude where the meter was engaged.\"},{\"name\":\"RateCodeID\",\"type\":\"int\",\"doc\":\"The final rate code in effect at the end of the trip. 1: Standard rate, 2:JFK, 3:Newark, 4:Nassau or Westchester, 5:Negotiated fare, 6:Group ride\"},{\"name\":\"store_and_fwd_flag\",\"type\":\"string\",\"doc\":\"This flag indicates whether the trip record was held in vehicle memory before sending to the vendor, aka “store and forward,” because the vehicle did not have a connection to the server. Y: store and forward trip N: not a store and forward trip\"},{\"name\":\"dropoff_longitude\",\"type\":\"double\",\"doc\":\"Longitude where the meter was disengaged.\"},{\"name\":\"dropoff_latitude\",\"type\":\"double\",\"doc\":\"Latitude where the meter was disengaged.\"},{\"name\":\"payment_type\",\"type\":\"int\",\"doc\":\"A numeric code signifying how the passenger paid for the trip. 1: Credit card 2: Cash 3: No charge 4: Dispute 5: Unknown 6: Voided trip\"},{\"name\":\"fare_amount\",\"type\":\"double\",\"doc\":\"The time-and-distance fare calculated by the meter.\"},{\"name\":\"extra\",\"type\":\"double\",\"doc\":\"Miscellaneous extras and surcharges. Currently, this only includes the $0.50 and $1 rush hour and overnight charges.\"},{\"name\":\"mta_tax\",\"type\":\"double\",\"doc\":\"$0.50 MTA tax that is automatically triggered based on the metered rate in use.\"},{\"name\":\"improvement_surcharge\",\"type\":\"double\",\"doc\":\"$0.30 improvement surcharge assessed trips at the flag drop. The improvement surcharge began being levied in 2015.\"},{\"name\":\"tip_amount\",\"type\":\"double\",\"doc\":\"Tip amount – This field is automatically populated for credit card tips. Cash tips are not included.\"},{\"name\":\"tolls_amount\",\"type\":\"double\",\"doc\":\"Total amount of all tolls paid in trip.\"},{\"name\":\"total_amount\",\"type\":\"double\",\"doc\":\"The total amount charged to passengers. Does not include cash tips.\"}]}",
  "valueSchemaInlined": "{\"type\":\"record\",\"name\":\"trip_record\",\"namespace\":\"com.landoop.transportation.nyc.trip.yellow\",\"doc\":\"Schema for yellow taxi trip records from NYC TLC data. [http://www.nyc.gov/html/tlc/html/about/trip_record_data.shtml]\",\"fields\":[{\"name\":\"VendorID\",\"type\":\"int\",\"doc\":\"A code indicating the TPEP provider that provided the record. 1: Creative Mobile Technologies, LLC 2: VeriFone Inc.\"},{\"name\":\"tpep_pickup_datetime\",\"type\":\"string\",\"doc\":\"The date and time when the meter was engaged.\"},{\"name\":\"tpep_dropoff_datetime\",\"type\":\"string\",\"doc\":\"The date and time when the meter was disengaged.\"},{\"name\":\"passenger_count\",\"type\":\"int\",\"doc\":\"The number of passengers in the vehicle. This is a driver-entered value.\"},{\"name\":\"trip_distance\",\"type\":\"double\",\"doc\":\"The elapsed trip distance in miles reported by the taximeter.\"},{\"name\":\"pickup_longitude\",\"type\":\"double\",\"doc\":\"Longitude where the meter was engaged.\"},{\"name\":\"pickup_latitude\",\"type\":\"double\",\"doc\":\"Latitude where the meter was engaged.\"},{\"name\":\"RateCodeID\",\"type\":\"int\",\"doc\":\"The final rate code in effect at the end of the trip. 1: Standard rate, 2:JFK, 3:Newark, 4:Nassau or Westchester, 5:Negotiated fare, 6:Group ride\"},{\"name\":\"store_and_fwd_flag\",\"type\":\"string\",\"doc\":\"This flag indicates whether the trip record was held in vehicle memory before sending to the vendor, aka “store and forward,” because the vehicle did not have a connection to the server. Y: store and forward trip N: not a store and forward trip\"},{\"name\":\"dropoff_longitude\",\"type\":\"double\",\"doc\":\"Longitude where the meter was disengaged.\"},{\"name\":\"dropoff_latitude\",\"type\":\"double\",\"doc\":\"Latitude where the meter was disengaged.\"},{\"name\":\"payment_type\",\"type\":\"int\",\"doc\":\"A numeric code signifying how the passenger paid for the trip. 1: Credit card 2: Cash 3: No charge 4: Dispute 5: Unknown 6: Voided trip\"},{\"name\":\"fare_amount\",\"type\":\"double\",\"doc\":\"The time-and-distance fare calculated by the meter.\"},{\"name\":\"extra\",\"type\":\"double\",\"doc\":\"Miscellaneous extras and surcharges. Currently, this only includes the $0.50 and $1 rush hour and overnight charges.\"},{\"name\":\"mta_tax\",\"type\":\"double\",\"doc\":\"$0.50 MTA tax that is automatically triggered based on the metered rate in use.\"},{\"name\":\"improvement_surcharge\",\"type\":\"double\",\"doc\":\"$0.30 improvement surcharge assessed trips at the flag drop. The improvement surcharge began being levied in 2015.\"},{\"name\":\"tip_amount\",\"type\":\"double\",\"doc\":\"Tip amount – This field is automatically populated for credit card tips. Cash tips are not included.\"},{\"name\":\"tolls_amount\",\"type\":\"double\",\"doc\":\"Total amount of all tolls paid in trip.\"},{\"name\":\"total_amount\",\"type\":\"double\",\"doc\":\"The total amount charged to passengers. Does not include cash tips.\"}]}",
  "valueSchemaVersion": 1,
  "valueType": "AVRO"
}
`

func TestNewDaatasetGroupCmdSuccess(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(datasetResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client
	
	cmd := NewDatasetGroupCmd()
	var outputValue string
	
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.NotEmpty(t, output)
}

func TestNewDatasetUpdateMetadataCmdSuccess(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(datasetResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client
	
	cmd := NewDatasetUpdateMetadataCmd()
	output, err := test.ExecuteCommand(cmd, 
		"--connection=kafka",
		"--name=topicName",
		"--description=Some Description",
	)

	assert.Nil(t, err)
	assert.Equal(t, "Lenses Metadata have been updated successfully\n", output)

	config.Client = nil
}

func TestNewDatasetUpdateMetadataCmdFailureNoConnection(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(datasetResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client
	
	cmd := NewDatasetUpdateMetadataCmd()
	output, err := test.ExecuteCommand(cmd,
		"--name=topicName",
		"--description=Some Description",
	)

	assert.NotEmpty(t, err)
	assert.Equal(t, "Usage:\n  updateMetadata [CONNECTION] [NAME] [DESCRIPTION] [flags]\n\nFlags:\n      --connection string    Name of the connection\n      --description string   Description of the dataset\n  -h, --help                 help for updateMetadata\n      --name string          Name of the dataset\n      --silent               run in silent mode. No printing info messages for CRUD except errors, defaults to false\n\n", output)

	config.Client = nil
}

func TestNewDatasetUpdateMetadataCmdFailureNoName(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(datasetResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client
	
	cmd := NewDatasetUpdateMetadataCmd()
	output, err := test.ExecuteCommand(cmd,
		"--connection=kafka",
		"--description=Some Description",
	)

	assert.NotEmpty(t, err)
	assert.Equal(t, "Usage:\n  updateMetadata [CONNECTION] [NAME] [DESCRIPTION] [flags]\n\nFlags:\n      --connection string    Name of the connection\n      --description string   Description of the dataset\n  -h, --help                 help for updateMetadata\n      --name string          Name of the dataset\n      --silent               run in silent mode. No printing info messages for CRUD except errors, defaults to false\n\n", output)

	config.Client = nil
}