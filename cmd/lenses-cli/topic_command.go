package main

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/landoop/lenses-go"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newTopicsGroupCommand())
	app.AddCommand(newTopicGroupCommand())
}

type topicView struct {
	lenses.Topic `yaml:",inline" header:"inline"`
	// for machine view-only.
	ValueSchema json.RawMessage `json:"valueSchema" yaml:"-"`
	KeySchema   json.RawMessage `json:"keySchema" yaml:"-"`
}

func newTopicView(cmd *cobra.Command, topic lenses.Topic) (t topicView) {
	t.Topic = topic

	// don't spend time here if we are not in the machine-friendly mode, table mode does not show so much details and couldn't be, schemas are big.
	if !bite.GetMachineFriendlyFlag(cmd) {
		return
	}

	if topic.KeySchema != "" {
		rawJSON, err := lenses.JSONAvroSchema(topic.KeySchema)
		if err != nil {
			return
		}

		if err = json.Unmarshal(rawJSON, &t.KeySchema); err != nil {
			return
		}
	}

	if topic.ValueSchema != "" {
		rawJSON, err := lenses.JSONAvroSchema(topic.ValueSchema)
		if err != nil {
			return
		}

		if err = json.Unmarshal(rawJSON, &t.ValueSchema); err != nil {
			return
		}
	}

	return
}

func newTopicsGroupCommand() *cobra.Command {
	var namesOnly, unwrap bool

	root := &cobra.Command{
		Use:           "topics",
		Short:         "List all available topics",
		Example:       "topics",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if namesOnly {
				topicNames, err := client.GetTopicsNames()
				if err != nil {
					return err
				}
				sort.Strings(topicNames)

				if unwrap {
					for _, name := range topicNames {
						fmt.Fprintln(cmd.OutOrStdout(), name)
					}
					return nil
				}

				// return printJSON(cmd, outlineStringResults("name", topicNames))
				return bite.PrintObject(cmd, bite.OutlineStringResults(cmd, "name", topicNames))
			}

			topics, err := client.GetTopics()
			if err != nil {
				return err
			}

			sort.Slice(topics, func(i, j int) bool {
				return topics[i].TopicName < topics[j].TopicName
			})

			topicsView := make([]topicView, len(topics))
			for i, topic := range topics {
				topicsView[i] = newTopicView(cmd, topic)
			}

			// return printJSON(cmd, topics)
			// lenses-cli topics --machine-friendly will print all information as JSON,
			// lenses-cli topics [--machine-friend=false] will print the necessary(struct fields tagged as "header") information as Table.
			return bite.PrintObject(cmd, topicsView, func(t lenses.Topic) bool {
				return !bite.GetMachineFriendlyFlag(cmd) && !t.IsControlTopic // on JSON we print everything.
			})
		},
	}

	root.Flags().BoolVar(&namesOnly, "names", false, "--names")
	root.Flags().BoolVar(&unwrap, "unwrap", false, "--unwrap")

	bite.CanPrintJSON(root)

	root.AddCommand(newTopicsMetadataSubgroupCommand())

	return root
}

type topicMetadataView struct {
	lenses.TopicMetadata `yaml:",inline" header:"inline"`
	ValueSchema          json.RawMessage `json:"valueSchema" yaml:"-"` // for view-only.
	KeySchema            json.RawMessage `json:"keySchema" yaml:"-"`   // for view-only.
}

func newTopicMetadataView(m lenses.TopicMetadata) (topicMetadataView, error) {
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
		Example:       "topics metadata",
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if topicName != "" {
				// view single.

				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to retrieve topic's metadata for '%s', it does not exist", topicName)
				meta, err := client.GetTopicMetadata(topicName)
				if err != nil {
					return err
				}

				viewMeta, err := newTopicMetadataView(meta)
				if err != nil {
					return err
				}

				// return printJSON(cmd, viewMeta)
				return bite.PrintObject(cmd, viewMeta)
			}

			meta, err := client.GetTopicsMetadata()
			if err != nil {
				return err
			}

			sort.Slice(meta, func(i, j int) bool {
				return meta[i].TopicName < meta[j].TopicName
			})

			viewMeta := make([]topicMetadataView, len(meta), len(meta))

			for i, m := range meta {
				viewMeta[i], err = newTopicMetadataView(m)
				if err != nil {
					return err
				}
			}

			// return printJSON(cmd, viewMeta)
			return bite.PrintObject(cmd, viewMeta)
		},
	}

	rootSub.Flags().StringVar(&topicName, "name", "", "--name=topicName if filled then it returns a single topic metadata for that specific topic")

	bite.CanPrintJSON(rootSub)

	rootSub.AddCommand(newTopicMetadataDeleteCommand())
	rootSub.AddCommand(newTopicMetadataCreateCommand())

	return rootSub
}

func newTopicMetadataDeleteCommand() *cobra.Command {
	var topicName string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a topic's metadata",
		Example:          `topics metadata delete --name="topicName"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": topicName}); err != nil {
				return err
			}

			if err := client.DeleteTopicMetadata(topicName); err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to delete, metadata for topic '%s' does not exist", topicName)
				return err
			}

			return bite.PrintInfo(cmd, "Metadata for topic '%s' deleted", topicName)
		},
	}

	cmd.Flags().StringVar(&topicName, "name", "", "--name=topicName")

	bite.CanBeSilent(cmd)

	return cmd
}

func newTopicMetadataCreateCommand() *cobra.Command {
	var meta lenses.TopicMetadata

	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or update an existing topic metadata",
		Example:          `topics metadata set ./topic_metadata.yml`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": meta.TopicName}); err != nil {
				return err
			}

			if err := client.CreateOrUpdateTopicMetadata(meta); err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Metadata for topic '%s' created", meta.TopicName)
		},
	}

	cmd.Flags().StringVar(&meta.TopicName, "name", "", "--name=topicName")
	cmd.Flags().StringVar(&meta.KeyType, "key-type", "", "--key-type=keyType")
	cmd.Flags().StringVar(&meta.ValueType, "value-type", "", "--value-type=valueType")
	bite.CanBeSilent(cmd)

	bite.Prepend(cmd, bite.FileBind(&meta))

	return cmd
}

func newTopicGroupCommand() *cobra.Command {
	var topicName string

	root := &cobra.Command{
		Use:              "topic",
		Short:            "Work with a particular topic based on the topic name, retrieve it or create a new one",
		Example:          `topic --name="existing_topic_name" or topic create --name="topic1" --replication=1 --partitions=1 --configs="{\"key\": \"value\"}"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": topicName}); err != nil {
				return err
			}

			// default is the retrieval of the particular topic info.
			topic, err := client.GetTopic(topicName)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "topic with name: '%s' does not exist", topicName)
				return err
			}

			return bite.PrintObject(cmd, newTopicView(cmd, topic))
		},
	}

	root.Flags().StringVar(&topicName, "name", "", "--name=topic1")
	bite.CanPrintJSON(root)

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
		Example:          `topic create --name="topic1" --replication=1 --partitions=1 --configs="{\"max.message.bytes\": \"1000010\"}"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": topic.TopicName}); err != nil {
				return err
			}

			if err := client.CreateTopic(topic.TopicName, topic.Replication, topic.Partitions, topic.Configs); err != nil {
				bite.FriendlyError(cmd, errResourceNotGoodMessage, "unable to create topic with name '%s', already exists", topic.TopicName)
				return err
			}

			return bite.PrintInfo(cmd, "Topic '%s' created", topic.TopicName)
		},
	}

	cmd.Flags().StringVar(&topic.TopicName, "name", "", "--name=topic1")
	cmd.Flags().IntVar(&topic.Replication, "replication", topic.Replication, "--relication=1")
	cmd.Flags().IntVar(&topic.Partitions, "partitions", topic.Partitions, "--partitions=1")
	cmd.Flags().StringVar(&configsRaw, "configs", "", `--configs="{\"max.message.bytes\": \"1000010\"}"`)
	bite.CanBeSilent(cmd)

	bite.ShouldTryLoadFile(cmd, &topic).Else(func() error { return bite.AllowEmptyFlag(bite.TryReadFile(configsRaw, &topic.Configs)) })
	// same
	// bite.Prepend(cmd, bite.FileBind(&topic, bite.ElseBind(func() error { return bite.AllowEmptyFlag(bite.TryReadFile(configsRaw, &topic.Configs)) })))

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
		Example:          `topic delete --name="topic1" [--partition=0 --offset=1260]`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": topicName}); err != nil {
				return err
			}

			if fromPartition >= 0 && toOffset >= 0 {
				// delete records.
				if err := client.DeleteTopicRecords(topicName, fromPartition, toOffset); err != nil {
					bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to delete records, topic '%s' does not exist", topicName)
					bite.FriendlyError(cmd, errResourceNotAccessibleMessage, "unable to delete records from topic '%s', not proper access", topicName)
					bite.FriendlyError(cmd, errResourceNotGoodMessage, "unable to delete records from topic '%s', invalid offset '%d' or partition '%d' passed", topicName, toOffset, fromPartition)
					return err
				}

				return bite.PrintInfo(cmd, "Records from topic '%s' and partition '%d' up to offset '%d', are marked for deletion. This may take a few moments to have effect", topicName, fromPartition, toOffset)
			}

			if err := client.DeleteTopic(topicName); err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to delete, topic '%s' does not exist", topicName)
				return err
			}

			return bite.PrintInfo(cmd, "Topic '%s' marked for deletion. This may take a few moments to have effect", topicName)
		},
	}

	cmd.Flags().StringVar(&topicName, "name", "", "--name=topic1")

	// negative default values because 0 is valid value.
	cmd.Flags().IntVar(&fromPartition, "partition", -1, "--partition=0 Deletes records from a specific partition (offset must set)")
	cmd.Flags().Int64Var(&toOffset, "offset", -1, "--offset=1260 Deletes records from a specific offset (partition must set)")
	bite.CanBeSilent(cmd)

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
		Example:          `topic update --name="topic1" --configs="[{\"key\": \"max.message.bytes\", \"value\": \"1000020\"}, ...]" or topic update ./topic.yml`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": topic.Name}); err != nil {
				return err
			}

			if err := client.UpdateTopic(topic.Name, topic.Configs); err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to update configs, topic '%s' does not exist", topic.Name)
				return err
			}

			return bite.PrintInfo(cmd, "Config updated for topic '%s'", topic.Name)
		},
	}

	cmd.Flags().StringVar(&topic.Name, "name", "", "--name=topic1")
	cmd.Flags().StringVar(&configsArrayRaw, "configs", "", `--configs="[{\"key\": \"max.message.bytes\", \"value\": \"1000020\"}, ...]"`)
	bite.CanBeSilent(cmd)

	bite.Prepend(cmd, bite.FileBind(&topic, bite.ElseBind(func() error { return bite.TryReadFile(configsArrayRaw, &topic.Configs) })))

	return cmd
}
