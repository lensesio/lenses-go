package imports

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewImportTopicsCommand creates `import topics` command
func NewImportTopicsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "topics",
		Short:            "topics",
		Example:          `import topics --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.TopicsPath)
			if err := loadTopics(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to load topics. [%s]", err.Error())
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
	files := utils.FindFiles(loadpath)
	topics, err := client.GetTopics()

	if err != nil {
		golog.Errorf("Error retrieving topics [%s]", err.Error())
		return err
	}

	for _, file := range files {
		var topic api.CreateTopicPayload
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &topic); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		found := false

		for _, lensesTopic := range topics {
			if lensesTopic.TopicName == topic.TopicName {
				found = true
				if err := client.UpdateTopic(topic.TopicName, []api.KV{topic.Configs}); err != nil {
					golog.Errorf("Error updating topic [%s]. [%s]", topic.TopicName, err.Error())
					return err
				}

				golog.Infof("Updated topic [%s]", topic.TopicName)
			}
		}

		if !found {
			if err := client.CreateTopic(topic.TopicName, topic.Replication, topic.Partitions, topic.Configs); err != nil {
				golog.Errorf("Error creating topic [%s]. [%s]", topic.TopicName, err.Error())
				return err
			}

			golog.Infof("Created topic [%s]", topic.TopicName)
		}
	}

	return nil
}
