package main

import (
	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTopicsCommand())
	rootCmd.AddCommand(newTopicGroupCommand())
}

func newTopicsCommand() *cobra.Command {
	var namesOnly bool

	cmd := cobra.Command{
		Use:           "topics",
		Short:         "List all available topic names",
		Example:       exampleString("topics"),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if namesOnly {
				topicNames, err := client.GetTopicsNames()
				if err != nil {
					return err
				}

				return printJSON(cmd.OutOrStdout(), outlineStringResults("name", topicNames))
			}

			topics, err := client.GetTopics()
			if err != nil {
				return err
			}

			return printJSON(cmd.OutOrStdout(), topics)
		},
	}

	cmd.Flags().BoolVar(&namesOnly, "names", false, "--names")
	cmd.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	cmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	return &cmd
}

func newTopicGroupCommand() *cobra.Command {
	var topicName string

	root := cobra.Command{
		Use:              "topic",
		Short:            "Work with a particular topic based on the topic name, retrieve it or create a new one",
		Example:          exampleString(`topic --name="existing_topic_name" or topic create --name="topic1" --replication=1 --partitions=1 --configs="{\"key\": \"value\"}"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// default is the retrieval of the particular topic info.
			topic, err := client.GetTopic(topicName)
			if err != nil {
				return err
			}

			return printJSON(cmd.OutOrStdout(), topic)
		},
	}

	root.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	root.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")
	root.Flags().StringVar(&topicName, "name", "", "--name=topic1")
	root.MarkFlagRequired("name")

	// subcommands
	root.AddCommand(newTopicCreateCommand())
	root.AddCommand(newTopicDeleteCommand())
	root.AddCommand(newTopicUpdateCommand())

	return &root
}

func newTopicCreateCommand() *cobra.Command {

	var (
		configsRaw string
		topic      = lenses.CreateTopicPayload{
			Replication: 1,
			Partitions:  1,
			Configs:     make(lenses.KV),
		}
	)

	cmd := cobra.Command{
		Use:              "create",
		Short:            "Creates a new topic",
		Example:          exampleString(`topic create --name="topic1" --replication=1 --partitions=1 --configs="{\"max.message.bytes\": \"1000010\"}"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// load from file.
				if err := loadFile(cmd, args[0], &topic); err != nil {
					return err
				}

			} else {
				// try load only the config from flag or file if possible.
				if err := tryReadFile(configsRaw, &topic.Configs); err != nil &&
					err.Error() != errFlagMissing.Error() { // allow empty.
					return err
				}
			}

			if err := checkRequiredFlags(flags{"name": topic.TopicName}); err != nil {
				return err
			}

			if err := client.CreateTopic(topic.TopicName, topic.Replication, topic.Partitions, topic.Configs); err != nil {
				return err
			}

			return echo(cmd, "Created topic %s", topic.TopicName)
		},
	}

	cmd.Flags().StringVar(&topic.TopicName, "name", "", "--name=topic1")
	cmd.Flags().BoolVar(&silent, "silent", false, `--silent`)
	cmd.Flags().IntVar(&topic.Replication, "replication", topic.Replication, "--relication=1")
	cmd.Flags().IntVar(&topic.Partitions, "partitions", topic.Partitions, "--partitions=1")

	// max.message.bytes has 1000012 is the default, which is the recommending maximum value
	// if we make it larger we may have fetch issues(?), so keep that in mind.
	cmd.Flags().StringVar(&configsRaw, "configs", "", `--configs="{\"max.message.bytes\": \"1000010\"}"`)

	return &cmd
}

func newTopicDeleteCommand() *cobra.Command {
	var topicName string

	cmd := cobra.Command{
		Use:              "delete",
		Short:            "Deletes a topic",
		Example:          exampleString(`topic delete --name="topic1"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.DeleteTopic(topicName); err != nil {
				return err
			}

			return echo(cmd, "Topic %s marked for deletion. This may take a few moments to have effect", topicName)
		},
	}

	cmd.Flags().StringVar(&topicName, "name", "", "--name=topic1")
	cmd.MarkFlagRequired("name")

	return &cmd
}

func newTopicUpdateCommand() *cobra.Command {
	var (
		configsArrayRaw string
		topic           = lenses.UpdateTopicPayload{
			Configs: make([]lenses.KV, 0),
		}
	)

	cmd := cobra.Command{
		Use:              "update",
		Short:            "Updates a topic's configs (as an array of config key-value map)",
		Example:          exampleString(`topic update --name="topic1" --configs="[{\"key\": \"max.message.bytes\", \"value\": \"1000020\"}, ...]" or topic update ./topic.yml`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				// load from file.
				if err := loadFile(cmd, args[0], &topic); err != nil {
					return err
				}
			} else {
				// load only the configs from file.
				if err := tryReadFile(configsArrayRaw, &topic.Configs); err != nil {
					return err
				}
			}

			if err := checkRequiredFlags(flags{"name": topic.Name}); err != nil {
				return err
			}

			if err := client.UpdateTopic(topic.Name, topic.Configs); err != nil {
				return err
			}

			return echo(cmd, "Configuration updated for topic %s", topic.Name)
		},
	}

	cmd.Flags().StringVar(&topic.Name, "name", "", "--name=topic1")
	cmd.Flags().StringVar(&configsArrayRaw, "configs", "", `--configs="[{\"key\": \"max.message.bytes\", \"value\": \"1000020\"}, ...]"`)

	return &cmd
}
