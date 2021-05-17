package topicsettings

import (
	"fmt"
	"os"

	"strings"

	"github.com/MakeNowJust/heredoc"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewTopicSettingsCmd returns the Kafka Topics Settings
func NewTopicSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "topic-settings",
		Long: heredoc.Doc(`
			View/Update the Topic Settings for Kafka Topics.

			View the settings for Kafka Topic. Settings restict tenants from overloading the Kafka Cluster(s).
			The settings can be imported/exported to different Kafka Cluster to allow for best GitOps practices.
			
			Note that you need to use either JSON or YAML format.
		`),
		Example: heredoc.Doc(`
			$ lenses-cli topic-settings --output="JSON"
			$ lenses-cli topic-settings --output="YAML"
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := config.Client.GetTopicSettings()
			if err != nil {
				return errors.Wrap(err, utils.RED("✘ Error"))
			}

			outputFlagValue := strings.ToUpper(bite.GetOutPutFlag(cmd))
			if outputFlagValue != "JSON" && outputFlagValue != "YAML" {
				fmt.Fprintln(os.Stderr, utils.YELLOW("! Plesese use JSON or YAML output to see the object\n"))
			}

			return bite.PrintObject(cmd, settings)
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, utils.Green("✓ Read Topic Settings"))
		},
	}

	cmd.AddCommand(UpdateTopicSettingsCmd())

	return cmd
}

// UpdateTopicSettingsCmd updates the Kafka Topics Settings
func UpdateTopicSettingsCmd() *cobra.Command {
	var settings api.TopicSettingsRequest
	var namingPattern string
	var namingDescription string

	cmd := &cobra.Command{
		Use: "update",
		Long: heredoc.Doc(`
			Update settings for Kafka Topics

			Update the Settings for Kafka, in order to restrict tennants from overloading the Kafka Cluster(s).
			Set sensible default for "Minimum and Maximum Partitions", "Minimum and Maximum Replication Factor" and 
			"Retention Time and Size".

			Note that "Partitions" and "Replication" are positive integers, and "Retention Time and Size" need to be set in
			Milliseconds and Bytes respectively or can be set to -1, to signify inifite retention.
		`),
		Example: heredoc.Doc(`
			$ lenses-cli topic-settings update --partitions-min=1 --replication-min=1 --retention-size-max=-1 --retention-time-max=-1
		`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if namingPattern == "" && namingDescription == "" {
				settings.Naming = nil
			}

			if namingPattern != "" && namingDescription != "" {
				var naming api.Naming

				naming.Description = namingDescription
				naming.Pattern = namingPattern
				settings.Naming = &naming
			}

			if namingPattern != "" && namingDescription == "" {
				return fmt.Errorf(utils.RED("'naming-description' is mandatory if `naming-pattern` is provided"))
			}

			if namingPattern == "" && namingDescription != "" {
				return fmt.Errorf(utils.RED("'naming-pattern' is mandatory if `naming-description` is provided"))
			}

			err := config.Client.UpdateTopicSettings(settings)
			return errors.Wrap(err, utils.RED("✘ Error"))
		},
		PostRun: func(cmd *cobra.Command, args []string) {
			fmt.Fprintln(os.Stderr, utils.Green("✓ Updated Topic Settings"))
		},
	}

	cmd.Flags().IntVar(&settings.Config.Partitions.Min, "partitions-min", 0, "The minimum number of partitions when creating a topic")
	cmd.Flags().IntVar(&settings.Config.Partitions.Max, "partitions-max", 0, "The maximum number of partitions when creating a topic")
	cmd.Flags().IntVar(&settings.Config.Replication.Min, "replication-min", 0, "The minimum number of replicas for each partition")
	cmd.Flags().IntVar(&settings.Config.Replication.Max, "replication-max", 0, "The maximum number of replicas for each partition")

	cmd.Flags().Int64Var(&settings.Config.Retention.Size.Default, "retention-size-default", -1, "Default retention size")
	cmd.Flags().Int64Var(&settings.Config.Retention.Size.Max, "retention-size-max", -1, "Maximum retention size")
	cmd.Flags().Int64Var(&settings.Config.Retention.Time.Default, "retention-time-default", -1, "Default retention time")
	cmd.Flags().Int64Var(&settings.Config.Retention.Time.Max, "retention-time-max", -1, "Maximum retention time")

	cmd.Flags().StringVar(&namingDescription, "naming-description", "", "Naming description")
	cmd.Flags().StringVar(&namingPattern, "naming-pattern", "", "Regex pattern")

	cmd.MarkFlagRequired("partitions-min")
	cmd.MarkFlagRequired("retention-min")

	cmd.MarkFlagRequired("retention-size-max")
	cmd.MarkFlagRequired("retention-time-max")

	return cmd
}
