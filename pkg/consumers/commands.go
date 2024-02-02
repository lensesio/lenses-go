package consumers

import (
	"errors"
	"fmt"
	"strconv"

	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/spf13/cobra"
)

const (
	consumersCmdDescLong string = "APIs for managing Kafka consumer groups."
	offsetsCmdDescLong   string = "APIs for managing Kafka consumer groups partition offsets."

	updateSingleCmdDescLong string = "Updates consumer group offsets for a single partition of a single topic."
	updateSingleCmdExample  string = `
  # Update a single topic's partition offset to the specified value
  update-single-partition --group <group_name> --topic <topic_name> --partition <partition_id> --to-offset <offset_id>

  # Update to the earliest offset available
  update-single-partition --group <group_name> --topic <topic_name> --partition <partition_id> --to-earliest

  # Update to the latest offset available
  update-single-partition --group <group_name> --topic <topic_name> --partition <partition_id> --to-latest`

	updateSingleCmdSuccess      string = "Update single topic partition offset has succeeded"
	updateSingleCmdFailure      string = "Update single topic partition offset has failed"
	updateSingleCmdMissingFlags string = "Missing flags, \"to-offset\" or \"to-earliest\" or \"to-latest\""

	updateMultipleCmdDescLong string = "Updates consumer group offsets for all partitions of multiple topics of a consumer group."
	updateMultipleCmdExample  string = `
  # Update offsets for all partitions of multiple topics of a consumer group to the specified timestamp (ISO 8601 forma)
  lenses-cli consumers offsets update-multiple-partitions --group <group_name> --topic <topic_name> --topic <topic_name> --to-datetime <datetime>

  # Update all partitions of multiple topics to the earliest offset available
  lenses-cli consumers offsets update-multiple-partitions --group <group_name> --topic <topic_name> --to-earliest

  # Update all partitions of multiple topics to the latest offset available
  lenses-cli consumers offsets update-multiple-partitions --group <group_name> --topic <topic_name> --to-latest`
	updateMultipleCmdSuccess string = "Bulk update offsets for a consumer group has succeeded"
	updateMultipleCmdFailure string = "Bulk update offsets for a consumer group has failed!"

	deleteSingleOffsetCmdDescLong string = "Deletes consumer group offsets for a single partition of a single topic."
	deleteSingleOffsetCmdSuccess  string = "Delete single topic partition offset has succeeded"
	deleteSingleOffsetCmdFailure  string = "Delete single topic partition offset has failed"
	deleteSingleOffsetCmdExample  string = `
	# Delete a single topic's partition offset
	lenses-cli consumers offsets delete-single-partition --group <group_name> --topic <topic_name> --partition <partition_id>`

	deleteMultipleOffsetsCmdDescLong string = "Deletes consumer group offsets for all partitions of multiple topics."
	deleteMultipleOffsetsCmdSuccess  string = "Delete single topic partition offset has succeeded"
	deleteMultipleOffsetsCmdFailure  string = "Delete single topic partition offset has failed"
	deleteMultipleOffsetsCmdExample  string = `
	# Update offsets for all partitions of multiple topics of a consumer group to the specified timestamp (ISO 8601 forma)
	lenses-cli consumers offsets delete-multiple-partitions --group <group_name> --topic <topic_name> --topic <topic_name>`

	deleteConsumerGroupCmdDescLong string = "Deletes a single consumer group."
	deleteConsumerGroupCmdSuccess  string = "Delete consumer group has succeeded"
	deleteConsumerGroupCmdFailure  string = "Delete consumer group has failed"
	deleteConsumerGroupCmdExample  string = `
	# Update offsets for all partitions of multiple topics of a consumer group to the specified timestamp (ISO 8601 forma)
	lenses-cli consumers delete-consumer-group --group <group_name>`
)

var errMultipleTopics = errors.New("Only one topic is allowed")
var errMissingSinglePartitionFlag = errors.New("required flags, \"to-offset\" or \"to-earliest\" or \"to-latest\" not set")
var errMissingMultiplePartitionsFlag = errors.New("required flags, \"to-datetime\" or \"to-earliest\" or \"to-latest\" not set")
var errTopicMissing = errors.New("required flag \"topic\" not set")
var errTopicsMissing = errors.New("required flags \"topic\" or \"all-topics\" not set")

// NewRootCommand registers the `consumers` subcommand to Cobra and returns it
func NewRootCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "consumers",
		Short:            consumersCmdDescLong,
		Long:             consumersCmdDescLong,
		TraverseChildren: true,
	}

	cmd.AddCommand(newOffsetsCommand())
	cmd.AddCommand(newConsumerGroupDelete())

	return cmd
}

func newOffsetsCommand() *cobra.Command {
	var (
		group string
		topic []string
	)
	cmd := &cobra.Command{
		Use:              "offsets",
		Long:             offsetsCmdDescLong,
		TraverseChildren: true,
	}

	cmd.PersistentFlags().StringVarP(&group, "group", "g", "", "Consumer Group ID")
	cmd.PersistentFlags().StringSliceVarP(&topic, "topic", "t", nil, "Topic")
	cmd.MarkPersistentFlagRequired("group")
	cmd.AddCommand(newOffsetsUpdateSinglePartition())
	cmd.AddCommand(newOffsetsUpdateMultiplePartition())
	cmd.AddCommand(newOffsetsDeleteSinglePartition())
	cmd.AddCommand(newOffsetsDeleteMultiplePartition())

	return cmd
}

func newOffsetsUpdateSinglePartition() *cobra.Command {
	var (
		partition, toOffset  int
		toEarliest, toLatest bool
	)
	cmd := &cobra.Command{
		Use:     "update-single-partition",
		Long:    updateSingleCmdDescLong,
		Example: updateSingleCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {

			if !cmd.Flags().Changed("topic") {
				fmt.Fprintln(cmd.ErrOrStderr(), errTopicMissing)
				return nil
			}
			topicsList, err := cmd.Flags().GetStringSlice("topic")

			if len(topicsList) > 1 {
				return errMultipleTopics
			}
			groupID, _ := cmd.Flags().GetString("group")
			topicName := topicsList[0]
			partitionID, _ := cmd.LocalFlags().GetInt("partition")
			offset := -100
			offsetType := ""

			localFlags := cmd.LocalFlags()
			if localFlags.Changed("to-offset") {
				offsetType = "absolute"
				offset, _ = cmd.LocalFlags().GetInt("to-offset")
			} else if localFlags.Changed("to-earliest") {
				offsetType = "start"
			} else if localFlags.Changed("to-latest") {
				offsetType = "end"
			} else {
				return errMissingSinglePartitionFlag
			}

			err = config.Client.UpdateSingleTopicOffset(groupID, topicName, strconv.Itoa(partitionID), offsetType, offset)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), updateSingleCmdFailure)
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), updateSingleCmdSuccess)
			return nil
		},
	}

	cmd.Flags().IntVar(&partition, "partition", -100, "The partition ID")
	cmd.Flags().IntVar(&toOffset, "to-offset", -100, "Reset to a specific offset value")
	cmd.Flags().BoolVar(&toEarliest, "to-earliest", false, "Reset to earliest offset possible")
	cmd.Flags().BoolVar(&toLatest, "to-latest", false, "Reset to latest offset possible")
	cmd.MarkFlagRequired("partition")

	return cmd
}

func newOffsetsDeleteSinglePartition() *cobra.Command {
	var (
		partition int
	)

	cmd := &cobra.Command{
		Use:     "delete-single-partition-offsets",
		Long:    deleteSingleOffsetCmdDescLong,
		Example: deleteSingleOffsetCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {

			if !cmd.Flags().Changed("topic") {
				fmt.Fprintln(cmd.ErrOrStderr(), errTopicMissing)
				return nil
			}
			topicsList, err := cmd.Flags().GetStringSlice("topic")

			if len(topicsList) > 1 {
				return errMultipleTopics
			}
			groupID, _ := cmd.Flags().GetString("group")
			topicName := topicsList[0]
			partitionID, _ := cmd.LocalFlags().GetInt("partition")

			err = config.Client.DeleteSingleTopicOffset(groupID, topicName, strconv.Itoa(partitionID))
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), deleteSingleOffsetCmdFailure)
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), deleteSingleOffsetCmdSuccess)
			return nil
		},
	}

	cmd.Flags().IntVar(&partition, "partition", -100, "The partition ID")
	cmd.MarkFlagRequired("partition")

	return cmd
}

func newOffsetsDeleteMultiplePartition() *cobra.Command {
	var (
		allTopics bool
	)
	cmd := &cobra.Command{
		Use:     "delete-multiple-topic",
		Long:    deleteMultipleOffsetsCmdDescLong,
		Example: deleteMultipleOffsetsCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var groupID string
			var topics []string

			groupID, _ = cmd.Flags().GetString("group")
			localFlags := cmd.LocalFlags()

			if cmd.Flags().Changed("topic") {
				topics, _ = cmd.Flags().GetStringSlice("topic")
			} else if localFlags.Changed("all-topics") {
				topics = nil
			} else {
				fmt.Fprintln(cmd.ErrOrStderr(), errTopicsMissing)
				return nil
			}

			err := config.Client.DeleteMultipleTopicsOffset(groupID, topics)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), deleteMultipleOffsetsCmdFailure)
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), deleteMultipleOffsetsCmdSuccess)
			return nil
		},
	}

	cmd.Flags().BoolVar(&allTopics, "all-topics", false, "Target all topics for this consumer group")

	return cmd
}

func newConsumerGroupDelete() *cobra.Command {
	var (
		group string
	)
	cmd := &cobra.Command{
		Use:     "delete-consumer-group",
		Long:    deleteConsumerGroupCmdDescLong,
		Example: deleteConsumerGroupCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {

			cmd.PersistentFlags().StringVarP(&group, "group", "g", "", "Consumer Group ID")
			cmd.MarkPersistentFlagRequired("group")

			var groupID string

			groupID, _ = cmd.Flags().GetString("group")

			err := config.Client.DeleteConsumerGroup(groupID)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), deleteConsumerGroupCmdFailure)
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), deleteConsumerGroupCmdSuccess)
			return nil
		},
	}

	return cmd
}

func newOffsetsUpdateMultiplePartition() *cobra.Command {
	var (
		toDatetime                      string
		toEarliest, toLatest, allTopics bool
	)
	cmd := &cobra.Command{
		Use:     "update-multiple-partitions",
		Long:    updateMultipleCmdDescLong,
		Example: updateMultipleCmdExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			var groupID string
			var topics []string
			var offsetType string
			var targetDate string

			groupID, _ = cmd.Flags().GetString("group")
			localFlags := cmd.LocalFlags()

			if cmd.Flags().Changed("topic") {
				topics, _ = cmd.Flags().GetStringSlice("topic")
			} else if localFlags.Changed("all-topics") {
				topics = nil
			} else {
				fmt.Fprintln(cmd.ErrOrStderr(), errTopicsMissing)
				return nil
			}

			if localFlags.Changed("to-datetime") {
				offsetType = "timestamp"
				targetDate, _ = cmd.LocalFlags().GetString("to-datetime")

			} else if localFlags.Changed("to-earliest") {
				offsetType = "start"
			} else if localFlags.Changed("to-latest") {
				offsetType = "end"
			} else {
				return errMissingMultiplePartitionsFlag
			}

			err := config.Client.UpdateMultipleTopicsOffset(groupID, offsetType, targetDate, topics)
			if err != nil {
				fmt.Fprintln(cmd.ErrOrStderr(), updateMultipleCmdFailure)
				return nil
			}

			fmt.Fprintln(cmd.OutOrStdout(), updateMultipleCmdSuccess)
			return nil
		},
	}

	cmd.Flags().StringVar(&toDatetime, "to-datetime", "", "The target timestamp in ISO 8601 format")
	cmd.Flags().BoolVar(&toEarliest, "to-earliest", false, "Reset to earliest offset possible")
	cmd.Flags().BoolVar(&toLatest, "to-latest", false, "Reset to latest offset possible")
	cmd.Flags().BoolVar(&allTopics, "all-topics", false, "Target all topics for this consumer group")

	return cmd
}
