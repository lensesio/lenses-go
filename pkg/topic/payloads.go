package topic

import (
	"encoding/json"
	"strings"

	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg/api"
	"github.com/spf13/cobra"
)

type topicMetadataView struct {
	api.TopicMetadata `yaml:",inline" header:"inline"`
	ValueSchema       json.RawMessage `json:"valueSchema" yaml:"-"` // for view-only.
	KeySchema         json.RawMessage `json:"keySchema" yaml:"-"`   // for view-only.
}

func newTopicView(cmd *cobra.Command, client *api.Client, topic api.Topic) (t topicView) {
	t.Topic = topic
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	// don't spend time here if we are not in the machine-friendly mode, table mode does not show so much details and couldn't be, schemas are big.
	if output != "JSON" && output != "YAML" {
		return
	}

	if topic.KeySchema != "" {
		rawJSON, err := api.JSONAvroSchema(topic.KeySchema)
		if err != nil {
			return
		}

		if err = json.Unmarshal(rawJSON, &t.KeySchema); err != nil {
			return
		}
	}

	if topic.ValueSchema != "" {
		rawJSON, err := api.JSONAvroSchema(topic.ValueSchema)
		if err != nil {
			return
		}

		if err = json.Unmarshal(rawJSON, &t.ValueSchema); err != nil {
			return
		}
	}

	return
}

func newTopicMetadataView(m api.TopicMetadata) (topicMetadataView, error) {
	viewM := topicMetadataView{m, nil, nil}

	if len(m.ValueSchemaRaw) > 0 {
		rawJSON, err := api.JSONAvroSchema(m.ValueSchemaRaw)
		if err != nil {
			return viewM, err
		}

		if err = json.Unmarshal(rawJSON, &viewM.ValueSchema); err != nil {
			return viewM, err
		}

		// clear raw (avro) values and keep only the jsoned(ValueSchema, KeySchema).
		viewM.ValueSchemaRaw = ""
	}

	if len(m.KeySchemaRaw) > 0 {
		rawJSON, err := api.JSONAvroSchema(m.KeySchemaRaw)
		if err != nil {
			return viewM, err
		}

		if err = json.Unmarshal(rawJSON, &viewM.KeySchema); err != nil {
			return viewM, err
		}

		viewM.KeySchemaRaw = ""
	}

	return viewM, nil
}
