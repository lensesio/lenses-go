package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newTopicsGroupCommand())
	rootCmd.AddCommand(newTopicGroupCommand())
}

func newTopicsGroupCommand() *cobra.Command {
	var namesOnly, noJSON bool

	root := &cobra.Command{
		Use:           "topics",
		Short:         "List all available topics",
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
			// TODO:
			// return printTable(cmd, topics, func(t lenses.Topic) bool {
			// 	return !t.IsControlTopic
			// })
		},
	}

	root.Flags().BoolVar(&namesOnly, "names", false, "--names")
	root.Flags().BoolVar(&noJSON, "no-json", false, "--no-json")

	canPrintJSON(root)

	root.AddCommand(newTopicsMetadataSubgroupCommand())

	return root
}

type topicMetadataView struct {
	lenses.TopicMetadata `yaml:",inline"`
	ValueSchema          json.RawMessage `json:"valueSchema" yaml:"-"` // for view-only.
	KeySchema            json.RawMessage `json:"keySchema" yaml:"-"`   // for view-only.
}

func newtopicMetadataView(m lenses.TopicMetadata) (topicMetadataView, error) {
	viewM := topicMetadataView{m, nil, nil}

	if len(m.ValueSchemaRaw) > 0 {
		rawJSON, err := lenses.JSONAvroSchema(m.ValueSchemaRaw)
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
		rawJSON, err := lenses.JSONAvroSchema(m.KeySchemaRaw)
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

func newTopicsMetadataSubgroupCommand() *cobra.Command {
	var topicName string

	rootSub := &cobra.Command{
		Use:           "metadata",
		Short:         "List all available topics metadata",
		Example:       exampleString("topics metadata"),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if topicName != "" {
				// view single.

				errResourceNotFoundMessage = fmt.Sprintf("unable to retrieve topic's metadata for '%s', it does not exist", topicName)
				meta, err := client.GetTopicMetadata(topicName)
				if err != nil {
					return err
				}

				viewMeta, err := newtopicMetadataView(meta)
				if err != nil {
					return err
				}

				return printJSON(cmd, viewMeta)
			}

			meta, err := client.GetTopicsMetadata()
			if err != nil {
				return err
			}

			viewMeta := make([]topicMetadataView, len(meta), len(meta))

			for i, m := range meta {
				viewMeta[i], err = newtopicMetadataView(m)
				if err != nil {
					return err
				}
			}

			return printJSON(cmd, viewMeta)
		},
	}

	rootSub.Flags().StringVar(&topicName, "name", "", "--name=topicName if filled then it returns a single topic metadata for that specific topic")

	canBeSilent(rootSub)
	canPrintJSON(rootSub)

	rootSub.AddCommand(newTopicMetadataDeleteCommand())
	rootSub.AddCommand(newTopicMetadataCreateCommand())

	return rootSub
}

func newTopicMetadataDeleteCommand() *cobra.Command {
	var topicName string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a topic's metadata",
		Example:          exampleString(`topics metadata delete --name="topicName"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": topicName}); err != nil {
				return err
			}

			if err := client.DeleteTopicMetadata(topicName); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to delete, metadata for topic '%s' does not exist", topicName)
				return err
			}

			return echo(cmd, "Metadata for topic '%s' deleted", topicName)
		},
	}

	cmd.Flags().StringVar(&topicName, "name", "", "--name=topicName")

	canBeSilent(cmd)

	return cmd
}

func newTopicMetadataCreateCommand() *cobra.Command {
	var meta lenses.TopicMetadata

	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or update an existing topic metadata",
		Example:          exampleString(`topics metadata set ./topic_metadata.yml`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": meta.TopicName}); err != nil {
				return err
			}

			if err := client.CreateOrUpdateTopicMetadata(meta); err != nil {
				return err
			}

			return echo(cmd, "Metadata for topic '%s' created", meta.TopicName)
		},
	}

	cmd.Flags().StringVar(&meta.TopicName, "name", "", "--name=topicName")
	cmd.Flags().StringVar(&meta.KeyType, "key-type", "", "--key-type=keyType")
	cmd.Flags().StringVar(&meta.ValueType, "value-type", "", "--value-type=valueType")

	canBeSilent(cmd)

	shouldTryLoadFile(cmd, &meta)

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
			// TODO: return printTable(cmd, topic)
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
		Short:            "Create a new topic",
		Example:          exampleString(`topic create --name="topic1" --replication=1 --partitions=1 --configs="{\"max.message.bytes\": \"1000010\"}"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": topic.TopicName}); err != nil {
				return err
			}

			if err := client.CreateTopic(topic.TopicName, topic.Replication, topic.Partitions, topic.Configs); err != nil {
				errResourceNotGoodMessage = fmt.Sprintf("unable to create topic with name '%s', already exists", topic.TopicName)
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
	var (
		topicName string
		// and for records with offset.
		fromPartition int
		toOffset      int64
	)

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a topic",
		Example:          exampleString(`topic delete --name="topic1" [--partition=0 --offset=1260]`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": topicName}); err != nil {
				return err
			}

			if fromPartition >= 0 && toOffset >= 0 {
				// delete records.
				if err := client.DeleteTopicRecords(topicName, fromPartition, toOffset); err != nil {
					errResourceNotFoundMessage = fmt.Sprintf("unable to delete records, topic '%s' does not exist", topicName)
					errResourceNotAccessibleMessage = fmt.Sprintf("unable to delete records from topic '%s', not proper access", topicName)
					errResourceNotGoodMessage = fmt.Sprintf("unable to delete records from topic '%s', invalid offset '%d' or partition '%d' passed", topicName, toOffset, fromPartition)
					return err
				}

				return echo(cmd, "Records from topic '%s' and partition '%d' up to offset '%d', are marked for deletion. This may take a few moments to have effect", topicName, fromPartition, toOffset)
			}

			if err := client.DeleteTopic(topicName); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to delete, topic '%s' does not exist", topicName)
				return err
			}

			return echo(cmd, "Topic %s marked for deletion. This may take a few moments to have effect", topicName)
		},
	}

	cmd.Flags().StringVar(&topicName, "name", "", "--name=topic1")

	// negative default values because 0 is valid value.
	cmd.Flags().IntVar(&fromPartition, "partition", -1, "--partition=0 Deletes records from a specific partition (offset must set)")
	cmd.Flags().Int64Var(&toOffset, "offset", -1, "--offset=1260 Deletes records from a specific offset (partition must set)")

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
		Short:            "Update a topic's configs (as an array of config key-value map)",
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
