package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportTopicsCommand creates `export topics` command
func NewExportTopicsCommand() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:              "topics",
		Short:            "export topics",
		Example:          `export topics --resource-name my-topic`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			if err := writeTopics(cmd, config.Client, name); err != nil {
				golog.Errorf("Error writing topics. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract dependencies, topics, acls, quotas, alerts")
	cmd.Flags().StringVar(&name, "resource-name", "", "The topic name to export")
	cmd.Flags().StringVar(&topicExclusions, "exclude", "", "Topics to exclude")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Topics with the prefix only")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeTopics(cmd *cobra.Command, client *api.Client, topicName string) error {
	var requests []api.CreateTopicPayload

	raw, err := client.GetTopics()

	if err != nil {
		return err
	}

	for _, topic := range raw {

		// don't export control topics
		excluded := false
		for _, exclude := range systemTopicExclusions {
			if strings.HasPrefix(topic.TopicName, exclude) ||
				strings.Contains(topic.TopicName, "KSTREAM-") ||
				strings.Contains(topic.TopicName, "_agg_") ||
				strings.Contains(topic.TopicName, "_sql_store_") {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		// exclude any user defined
		excluded = false
		for _, exclude := range strings.Split(topicExclusions, ",") {
			if topic.TopicName == exclude {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		if prefix != "" && !strings.HasPrefix(topic.TopicName, prefix) {
			continue
		}

		if topicName != "" && topicName == topic.TopicName {
			overrides := getTopicConfigOverrides(topic.Configs)
			request := topic.GetTopicAsRequest(overrides)
			return writeTopicsAsRequest(cmd, []api.CreateTopicPayload{request})
		}

		overrides := getTopicConfigOverrides(topic.Configs)
		requests = append(requests, topic.GetTopicAsRequest(overrides))
	}

	return writeTopicsAsRequest(cmd, requests)
}

func writeTopicsAsRequest(cmd *cobra.Command, requests []api.CreateTopicPayload) error {
	// write topics
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	for _, topic := range requests {

		fileName := fmt.Sprintf("topic-%s.%s", strings.ToLower(topic.TopicName), strings.ToLower(output))

		if err := utils.WriteFile(landscapeDir, pkg.TopicsPath, fileName, output, topic); err != nil {
			return err
		}
	}

	return nil
}

func getTopicConfigOverrides(configs []api.KV) api.KV {
	overrides := make(api.KV)

	for _, kv := range configs {
		if val, ok := kv["isDefault"]; ok {
			if val.(bool) == false {
				var name, value string

				if val, ok := kv["name"]; ok {
					name = val.(string)
				}

				if val, ok := kv["originalValue"]; ok {
					value = val.(string)
				}
				overrides[name] = value
			}
		}
	}

	return overrides
}
