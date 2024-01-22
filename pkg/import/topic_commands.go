package imports

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewImportTopicsCommand creates `import topics` command
func NewImportTopicsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "topics",
		Short:            "topics",
		Example:          `import topics --dir /my-landscape`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path = fmt.Sprintf("%s/%s", path, pkg.TopicsPath)
			if err := loadTopics(config.Client, cmd, path); err != nil {
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func loadTopics(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading topics from [%s]", loadpath)

	remoteTopics, err := client.GetTopics()
	if err != nil {
		golog.Errorf("Error retrieving topics [%s]", err.Error())
		return err
	}

	// create a map out of Lenses existing topics where key is topic name
	// and value is a custom object of the topic's partition num and configs that is simplyfied for comparing purposes
	// i.e. from a slice of maps to a single map where key is the config name and value is its original value
	// (all other keys are disregarded, e.g. "defaultValue", "documentation", "isDefault", etc.)
	type simplyfiedTopicPayload struct {
		partitions int
		configs    map[string]string
	}
	simplyfiedRemoteTopics := make(map[string]simplyfiedTopicPayload)

	for _, topic := range remoteTopics {
		configMap := make(map[string]string)
		for _, conf := range topic.Configs {
			configMap[fmt.Sprintf("%v", conf["name"])] = fmt.Sprintf("%v", conf["originalValue"])
		}
		simplyfiedRemoteTopics[topic.TopicName] = simplyfiedTopicPayload{
			topic.Partitions,
			configMap,
		}
	}

	files, err := utils.FindFiles(loadpath)
	if err != nil {
		return err
	}
	for _, file := range files {
		var topicFromFile api.CreateTopicPayload
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &topicFromFile); err != nil {
			return err
		}

		// if target imported topic exists then compare it with the instance found on the server
		if topicValue, ok := simplyfiedRemoteTopics[topicFromFile.TopicName]; ok {

			// compare the partition values
			if topicValue.partitions != topicFromFile.Partitions {
				if err := client.UpdateTopicPartitions(topicFromFile.TopicName, topicFromFile.Partitions); err != nil {
					return err
				}

				golog.Infof("Updated topic '%s' partitions with new value '%v'", topicFromFile.TopicName, topicFromFile.Partitions)
			}

			// compare the config from imported file with the config on the remote server
			for k, v := range topicFromFile.Configs {
				// If at least one config value is different then perform a single PUT on all config
				if v != topicValue.configs[k] {
					if err := client.UpdateTopicConfig(topicFromFile.TopicName, []api.KV{topicFromFile.Configs}); err != nil {
						return err
					}

					golog.Infof("Updated topic '%s' config", topicFromFile.TopicName)
					break
				}
			}
		} else {
			// If topic doesn't exist on the remote server then import it as new
			if err := client.CreateTopic(topicFromFile.TopicName, topicFromFile.Replication, topicFromFile.Partitions, topicFromFile.Configs); err != nil {
				return err
			}

			golog.Infof("Created topic [%s]", topicFromFile.TopicName)
		}
	}

	return nil
}
