package alert

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

//NewAlertGroupCommand creates the `alert` command
func NewAlertGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "alert",
		Short:            "Manage alerts",
		Example:          "alert",
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	root.AddCommand(
		NewGetAlertSettingsCommand(),
		NewAlertSettingGroupCommand(),
	)

	return root
}

//NewGetAlertsCommand creates the `alerts` command
func NewGetAlertsCommand() *cobra.Command {
	var (
		pageSize int
	)

	cmd := &cobra.Command{
		Use:              "alerts",
		Short:            "Print the registered alerts",
		Example:          "alerts",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			alerts, err := config.Client.GetAlerts(pageSize)
			if err != nil {
				return fmt.Errorf("failed to retrieve alerts. Error: [%s]", err.Error())
			}
			return bite.PrintObject(cmd, alerts)
		},
	}

	cmd.Flags().IntVar(&pageSize, "page-size", 25, "Size of items to be included in the list")

	bite.CanPrintJSON(cmd)

	cmd.AddCommand(DeleteAlertEventsCommand())

	return cmd
}

//DeleteAlertEventsCommand  creates the `alerts delete` command
func DeleteAlertEventsCommand() *cobra.Command {
	var olderThanTimestamp int64

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete all the alert events older than specified timestamp",
		Example:          "alerts delete --timestamp=1621244454127",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"timestamp": olderThanTimestamp}); err != nil {
				return err
			}

			if err := config.Client.DeleteAlertEvents(olderThanTimestamp); err != nil {
				return fmt.Errorf("Failed to delete alert events. [%s]", err.Error())
			}
			return bite.PrintInfo(cmd, "Alert events older than timestamp: [%d] deleted.", olderThanTimestamp)
		},
	}

	cmd.Flags().Int64Var(&olderThanTimestamp, "timestamp", 0, "All the alert events older than that timestamp will be removed.")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

//NewGetAlertSettingsCommand creates the `alert settings` command
func NewGetAlertSettingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "settings",
		Short:            "Print all alert settings",
		Example:          "alert settings",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := config.Client.GetAlertSettings()
			if err != nil {
				return fmt.Errorf("failed to retrieve alerts' settings. Error: [%s]", err.Error())
			}

			// force json, may contains conditions that are easier to be seen in json format.
			return bite.PrintJSON(cmd, settings)
		},
	}

	bite.CanPrintJSON(cmd)

	return cmd
}

//NewAlertSettingGroupCommand creates the `alert setting` command
func NewAlertSettingGroupCommand() *cobra.Command {
	var (
		id     int
		enable bool
	)

	cmd := &cobra.Command{
		Use:              "setting",
		Short:            "Print an alert's settings",
		Example:          "alert setting --id=1001",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if cmd.Flags().Changed("enable") {
				if err := config.Client.EnableAlertSetting(id, enable); err != nil {
					return fmt.Errorf("failed to enable an alert's condition. Error: [%s]", err.Error())
				}

				if enable {
					return bite.PrintInfo(cmd, "Alert setting [%d] enabled", id)
				}
				return bite.PrintInfo(cmd, "Alert setting [%d] disabled", id)
			}

			settings, err := config.Client.GetAlertSetting(id)
			if err != nil {
				return fmt.Errorf("failed to retrieve alert's settings. Error: [%s]", err.Error())
			}

			return bite.PrintObject(cmd, settings)
		},
	}

	cmd.Flags().IntVar(&id, "id", 0, "--id=1001")
	cmd.MarkFlagRequired("id")

	cmd.Flags().BoolVar(&enable, "enable", false, "--enable")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)

	cmd.AddCommand(NewUpdateAlertSettingsCommand())
	cmd.AddCommand(NewGetAlertSettingConditionsCommand())
	cmd.AddCommand(NewAlertSettingConditionGroupCommand())

	return cmd
}

// NewUpdateAlertSettingsCommand updates an alert's settings, e.g. channels, etc.
func NewUpdateAlertSettingsCommand() *cobra.Command {
	var alertSettings api.AlertSettingsPayload

	cmd := &cobra.Command{
		Use:              "set",
		Short:            "Update an alert's settings or load from file. If `enable` parameter omitted, it defaults to true",
		Example:          "alert setting set --id=1001 --enable=true --channel='b83c862c-7e23-4fb4-863d-c03e04102f90' or alert setting set ./alert_sett.yml`",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if alertSettings.AlertID == "" {
				return errors.New("requires `id` parameter")
			}
			if alertSettings.Channels == nil {
				return errors.New("requires `channels` parameter")
			}

			err := config.Client.UpdateAlertSettings(alertSettings)
			if err != nil {
				return fmt.Errorf("failed to update an alert's settings. Error: [%s]", err.Error())
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Update alert's setting has succeeded")
			return nil
		},
	}

	cmd.Flags().BoolVar(&alertSettings.Enable, "enable", true, "Whether to enable a given alert's channel(s)")
	cmd.Flags().StringArrayVar(&alertSettings.Channels, "channels", nil, "Channel UIDs")
	cmd.Flags().StringVar(&alertSettings.AlertID, "id", "", "Alert ID")

	bite.Prepend(cmd, bite.FileBind(&alertSettings))

	return cmd
}

//NewGetAlertSettingConditionsCommand creates `alert setting conditions`
func NewGetAlertSettingConditionsCommand() *cobra.Command {
	var alertID int

	cmd := &cobra.Command{
		Use:              "conditions",
		Short:            "Print alert setting's conditions",
		Example:          "alert setting conditions --alert=1001",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conds, err := config.Client.GetAlertSettingConditions(alertID)
			if err != nil {
				return fmt.Errorf("failed to retrieve alerts' setting conditions. Error: [%s]", err.Error())
			}

			return bite.PrintObject(cmd, conds)
		},
	}

	cmd.Flags().IntVar(&alertID, "alert", 0, "--alert=1001")
	cmd.MarkFlagRequired("alert")

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)

	return cmd
}

//NewAlertSettingConditionGroupCommand creates `alert setting condition`
func NewAlertSettingConditionGroupCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "condition",
		Short:            "Manage alert setting's condition",
		Example:          `alert setting condition delete --alert=1001 --condition="28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`,
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	rootSub.AddCommand(NewSetAlertSettingConditionCommand())
	rootSub.AddCommand(NewDeleteAlertSettingConditionCommand())

	return rootSub
}

//NewSetAlertSettingConditionCommand creates `alert condition set` command
func NewSetAlertSettingConditionCommand() *cobra.Command {

	var (
		conds     SettingConditionPayloads
		cond      SettingConditionPayload
		threshold api.Threshold
	)

	cmdExample := `
# Create
alert setting condition set --alert=2000 --group=<consumerGroupID> --topic=<topicName> --threshold=<lag> --mode=<PerPartitionMode/PerTopicMode>"

# Update
alert setting condition set --alert=<id> --group=<consumerGroupID> --topic=<topicName> --threshold=<lag> --mode=<PerPartitionMode/PerTopicMode> --conditionID=<conditionID> --channels=<channelID> --channels=<channelID>

# Producer type of alert category
lenses-cli alert setting condition set --alert=5000 --topic=my-topic --duration=PT6H --more-than=5

# optional channels can be used more than once
lenses-cli alert setting condition set --alert=5000 --topic=my-topic --duration=PT6H --more-than=5 \
	--channels="9176c428-be0e-4c36-aa1e-8b8a66782232" \
	--channels="7b184b13-f37c-4eee-9c0a-17dec2fd7bf5"

# Using YAML
alert setting condition set ./alert_cond.yml`

	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or Update an alert setting's condition or load from file",
		Example:          cmdExample,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			alertID := cond.AlertID

			switch alertID {
			case 5000: // Handling of Producer's type of alert settings condition
				if cond.Topic == "" {
					return errors.New(`required flag "topic" not set`)
				}

				if cond.MoreThan == 0 && cond.LessThan == 0 {
					return errors.New(`required flag "more-than" or "less-than" not set`)
				}

				if cond.MoreThan > 0 && cond.LessThan > 0 {
					return errors.New(`only one flag of "more-than" or "less-than" is supported`)
				}

				if cond.Duration == "" {
					return errors.New(`required flag "duration" not set`)
				}

				if cond.MoreThan != 0 {
					if cond.MoreThan <= 0 {
						return errors.New(`"more-than" flag should be greater than zero`)
					}
					threshold.Type = "more_than"
					threshold.Messages = cond.MoreThan
				}

				if cond.LessThan != 0 {
					if cond.LessThan <= 0 {
						return errors.New(`"less-than" flag should be greater than zero`)
					}
					threshold.Type = "less_than"
					threshold.Messages = cond.LessThan
				}

				err := config.Client.SetAlertSettingsProducerCondition(strconv.Itoa(alertID), cond.ConditionID, cond.Topic, threshold, cond.Duration, cond.Channels)
				if err != nil {
					return fmt.Errorf("failed to create or update an alert's setting conditions. Error: [%s]", err.Error())
				}
				if cond.ConditionID != "" {
					fmt.Fprintln(cmd.OutOrStdout(), "rule with condition ID \""+cond.ConditionID+"\" updated successfully")
					return nil
				}

				fmt.Fprintln(cmd.OutOrStdout(), "new rule created successfully")
				return nil
			default:
				if len(conds.Conditions) > 0 {
					alertID := conds.AlertID
					for _, condition := range conds.Conditions {
						err := config.Client.CreateAlertSettingsCondition(strconv.Itoa(alertID), condition, []string{})
						if err != nil {
							return fmt.Errorf("failed to create or update an alert's condition. Error: [%s]", err.Error())
						}
						bite.PrintInfo(cmd, "Condition [id=%d] added", alertID)
					}
					return nil
				}

				if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"alert": cond.AlertID}); err != nil {
					return errors.New(`required flag "alert" not set`)
				}

				if cond.Topic == "" {
					return errors.New(`required flag "topic" not set`)
				}

				if cond.Group == "" {
					return errors.New(`required flag "group" not set`)
				}

				if cond.Threshold <= 0 {
					return errors.New(`required flag "threshold" not a positive number`)
				}

				var parsedMode api.ConsumerConditionMode

				if cond.Mode == "PerTopicMode" {
					parsedMode = api.PerTopicMode
				} else if cond.Mode == "PerPartitionMode" {
					parsedMode = api.PerPartitionMode
				} else {
					return errors.New(`required flag "mode" not correct (supported modes: PerPartitionMode or PerTopicMode)`)
				}

				var channels = cond.Channels
				if channels == nil {
					channels = []string{}
				}

				payload := api.ConsumerAlertConditionRequestv1{
					Condition: api.ConsumerConditionDsl{
						Group:     cond.Group,
						Threshold: cond.Threshold,
						Topic:     cond.Topic,
						Mode:      parsedMode,
					},
					Channels: channels,
				}

				err := config.Client.SetAlertSettingsConsumerCondition(strconv.Itoa(cond.AlertID), cond.ConditionID, payload)
				if err != nil {
					golog.Errorf("Failed to creating/updating alert setting condition [%s]. [%s]", payload, err.Error())
					return fmt.Errorf("failed to create or update an alert's condition. Error: [%s]", err.Error())
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Create / Update rule with channels attached succeeded")

				return bite.PrintInfo(cmd, "Condition [id=%d] added", cond.AlertID)
			}
		},
	}

	cmd.Flags().IntVar(&cond.AlertID, "alert", 0, "Alert ID")
	cmd.Flags().StringVar(&cond.ConditionID, "conditionID", "", "Alert condition ID")
	cmd.Flags().StringArrayVar(&cond.Channels, "channels", nil, "Channel UIDs")
	cmd.Flags().StringVar(&cond.Topic, "topic", "", "Topic name")

	// Flags for "Consumer lag" alert category
	cmd.Flags().StringVar(&cond.Group, "group", "", "Group ID")
	cmd.Flags().IntVar(&cond.Threshold, "threshold", 0, "Threshold value of acceptable lag")
	cmd.Flags().StringVar(&cond.Mode, "mode", "", "Alert mode (PerPartitionMode/PerTopicMode)")

	// Flags for "Producers" alert category
	cmd.Flags().IntVar(&cond.MoreThan, "more-than", 0, "Threshold value of messages")
	cmd.Flags().IntVar(&cond.LessThan, "less-than", 0, "Threshold value of messages")
	cmd.Flags().StringVar(&cond.Duration, "duration", "", "ISO_8601 duration string - e.g. 1 minute = “PT1M”, 6 hours = “PT6H")

	bite.CanBeSilent(cmd)

	bite.Prepend(cmd, bite.FileBind(&cond))
	bite.Prepend(cmd, bite.FileBind(&conds))

	return cmd
}

//NewDeleteAlertSettingConditionCommand creates `alert condition delete` command
func NewDeleteAlertSettingConditionCommand() *cobra.Command {
	var (
		alertID       int
		conditionUUID string
	)

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an alert setting's condition",
		Example:          `alert setting condition delete --alert=1001 --condition="28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := config.Client.DeleteAlertSettingCondition(alertID, conditionUUID)
			if err != nil {
				return fmt.Errorf("failed to delete an alert's setting condition. Error: [%s]", err.Error())
			}

			return bite.PrintInfo(cmd, "Condition [%s] for alert setting [%d] deleted", conditionUUID, alertID)
		},
	}

	cmd.Flags().IntVar(&alertID, "alert", 0, "Alert ID")
	cmd.MarkFlagRequired("alert")
	cmd.Flags().StringVar(&conditionUUID, "condition", "", `Alert condition uuid .e.g. "28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`)
	cmd.MarkFlagRequired("condition")
	bite.CanBeSilent(cmd)

	return cmd
}

// NewGetAlertChannelTemplatesCommand creates the `alertchannel-templates` sub-command
func NewGetAlertChannelTemplatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "alertchannel-templates",
		Short: "List alert channel templates",
		Example: `
# List all alert channel templates
alertchannel-templates

# Do a simple query using jq
alertchannel-templates --output=json | jq '.[] | select(.name =="Slack")' 
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			alertChannelTemplates, err := config.Client.GetAlertChannelTemplates()
			if err != nil {
				return fmt.Errorf("failed to retrieve alert channel templates. [%s]", err.Error())
			}

			outputFlagValue := strings.ToUpper(bite.GetOutPutFlag(cmd))
			if outputFlagValue != "JSON" && outputFlagValue != "YAML" {
				bite.PrintInfo(cmd, "Info: use JSON or YAML output to get the complete object\n\n")
			}

			return bite.PrintObject(cmd, alertChannelTemplates)
		},
	}

	bite.CanPrintJSON(cmd)

	return cmd
}
