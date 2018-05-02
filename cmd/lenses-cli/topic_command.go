package main

import (
	"fmt"
	"sort"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTopicsCommand())
	rootCmd.AddCommand(newTopicGroupCommand())
}

func newTopicsCommand() *cobra.Command {
	var namesOnly, noJSON bool

	cmd := &cobra.Command{
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

				if noJSON {
					sort.Strings(topicNames)
					for _, name := range topicNames {
						fmt.Fprintln(cmd.OutOrStdout(), name)
					}
					return nil
				}

				return printJSON(cmd, outlineStringResults("name", topicNames))
			}

			topics, err := client.GetTopics()
			if err != nil {
				return err
			}

			sort.Slice(topics, func(i, j int) bool {
				return topics[i].TopicName < topics[j].TopicName
			})

			return printJSON(cmd, topics)
		},
	}

	cmd.Flags().BoolVar(&namesOnly, "names", false, "--names")
	cmd.Flags().BoolVar(&noJSON, "no-json", false, "--no-json")
	canPrintJSON(cmd)

	return cmd
}

func newTopicGroupCommand() *cobra.Command {
	var topicName string

	root := &cobra.Command{
		Use:              "topic",
		Short:            "Work with a particular topic based on the topic name, retrieve it or create a new one",
		Example:          exampleString(`topic --name="existing_topic_name" or topic create --name="topic1" --replication=1 --partitions=1 --configs="{\"key\": \"value\"}"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": topicName}); err != nil {
				return err
			}

			// default is the retrieval of the particular topic info.
			topic, err := client.GetTopic(topicName)
			if err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("topic with name: '%s' does not exist", topicName)
				return err
			}

			return printJSON(cmd, topic)
		},
	}

	root.Flags().StringVar(&topicName, "name", "", "--name=topic1")
	canPrintJSON(root)

	// subcommands
	root.AddCommand(newTopicCreateCommand())
	root.AddCommand(newTopicDeleteCommand())
	root.AddCommand(newTopicUpdateCommand())

	return root
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

	cmd := &cobra.Command{
		Use:              "create",
		Short:            "Creates a new topic",
		Example:          exampleString(`topic create --name="topic1" --replication=1 --partitions=1 --configs="{\"max.message.bytes\": \"1000010\"}"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": topic.TopicName}); err != nil {
				return err
			}

			if err := client.CreateTopic(topic.TopicName, topic.Replication, topic.Partitions, topic.Configs); err != nil {
				return err
			}

			return echo(cmd, "Topic '%s' created", topic.TopicName)
		},
	}

	cmd.Flags().StringVar(&topic.TopicName, "name", "", "--name=topic1")
	cmd.Flags().IntVar(&topic.Replication, "replication", topic.Replication, "--relication=1")
	cmd.Flags().IntVar(&topic.Partitions, "partitions", topic.Partitions, "--partitions=1")

	cmd.Flags().StringVar(&configsRaw, "configs", "", `--configs="{\"max.message.bytes\": \"1000010\"}"`)
	canBeSilent(cmd)

	shouldTryLoadFile(cmd, &topic).Else(func() error { return allowEmptyFlag(tryReadFile(configsRaw, &topic.Configs)) })

	return cmd
}

func newTopicDeleteCommand() *cobra.Command {
	var topicName string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Deletes a topic",
		Example:          exampleString(`topic delete --name="topic1"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": topicName}); err != nil {
				return err
			}

			if err := client.DeleteTopic(topicName); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to delete, topic '%s' does not exist", topicName)
				return err
			}

			return echo(cmd, "Topic %s marked for deletion. This may take a few moments to have effect", topicName)
		},
	}

	cmd.Flags().StringVar(&topicName, "name", "", "--name=topic1")
	canBeSilent(cmd)

	return cmd
}

func newTopicUpdateCommand() *cobra.Command {
	var (
		configsArrayRaw string
		topic           = lenses.UpdateTopicPayload{
			Configs: make([]lenses.KV, 0),
		}
	)

	cmd := &cobra.Command{
		Use:              "update",
		Short:            "Updates a topic's configs (as an array of config key-value map)",
		Example:          exampleString(`topic update --name="topic1" --configs="[{\"key\": \"max.message.bytes\", \"value\": \"1000020\"}, ...]" or topic update ./topic.yml`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": topic.Name}); err != nil {
				return err
			}

			if err := client.UpdateTopic(topic.Name, topic.Configs); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to update configs, topic '%s' does not exist", topic.Name)
				return err
			}

			return echo(cmd, "Configuration updated for topic %s", topic.Name)
		},
	}

	cmd.Flags().StringVar(&topic.Name, "name", "", "--name=topic1")
	cmd.Flags().StringVar(&configsArrayRaw, "configs", "", `--configs="[{\"key\": \"max.message.bytes\", \"value\": \"1000020\"}, ...]"`)
	canBeSilent(cmd)

	shouldTryLoadFile(cmd, &topic).Else(func() error { return tryReadFile(configsArrayRaw, &topic.Configs) })

	return cmd
}
