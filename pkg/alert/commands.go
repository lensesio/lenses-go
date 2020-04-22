package alert

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
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
		sse      bool
		pageSize int
	)

	cmd := &cobra.Command{
		Use:              "alerts",
		Short:            "Print the registered alerts",
		Example:          "alerts",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sse {
				handler := func(alert api.Alert) error {
					return bite.PrintObject(cmd, alert) // keep json here?
				}
				return config.Client.GetAlertsLive(handler)
			}
			alerts, err := config.Client.GetAlerts(pageSize)
			if err != nil {
				golog.Errorf("Failed to retrieve alerts. [%s]", err.Error())
				return err
			}
			return bite.PrintObject(cmd, alerts)
		},
	}

	cmd.Flags().BoolVar(&sse, "live", false, "Enables real-time push alert notifications")
	cmd.Flags().IntVar(&pageSize, "page-size", 25, "Size of items to be included in the list")

	bite.CanPrintJSON(cmd)

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
				return err
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
		id         int
		mustEnable bool
	)

	root := &cobra.Command{
		Use:              "setting",
		Short:            "Print an alert's settings",
		Example:          "alert setting --id=1001",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if mustEnable {
				if err := config.Client.EnableAlertSetting(id, mustEnable); err != nil {
					return err
				}

				return bite.PrintInfo(cmd, "Alert setting [%d] enabled", id)
			}

			settings, err := config.Client.GetAlertSetting(id)
			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, settings)
		},
	}

	root.Flags().IntVar(&id, "id", 0, "--id=1001")
	root.MarkFlagRequired("id")

	root.Flags().BoolVar(&mustEnable, "enable", false, "--enable")

	bite.CanPrintJSON(root)
	bite.CanBeSilent(root)

	root.AddCommand(NewUpdateAlertSettingsCommand())
	root.AddCommand(NewGetAlertSettingConditionsCommand())
	root.AddCommand(NewAlertSettingConditionGroupCommand())

	return root
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
				return err
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
				return err
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

	rootSub.AddCommand(NewCreateOrUpdateAlertSettingConditionCommand())
	rootSub.AddCommand(NewDeleteAlertSettingConditionCommand())

	return rootSub
}

//NewCreateOrUpdateAlertSettingConditionCommand creates `alert condition set` command
func NewCreateOrUpdateAlertSettingConditionCommand() *cobra.Command {
	var conds SettingConditionPayloads
	var cond SettingConditionPayload
	cmdExample := "\n# Create\nalert setting condition set --alert=1001 --condition=\"lag >= 100000\"\n" +
		"# Update\nalert setting condition set --alert <id> --condition=<condition> --conditionID=<conditionID> --channels=<channelID> --channels=<channelID>\n" +
		"# Using YAML\nalert setting condition set ./alert_cond.yml"

	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or Update an alert setting's condition or load from file",
		Example:          cmdExample,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(conds.Conditions) > 0 {
				alertID := conds.AlertID
				for _, condition := range conds.Conditions {
					err := config.Client.CreateOrUpdateAlertSettingCondition(alertID, condition)
					if err != nil {
						golog.Errorf("Failed to creating/updating alert setting condition [%s]. [%s]", condition, err.Error())
						return err
					}
					bite.PrintInfo(cmd, "Condition [id=%d] added", alertID)
				}
				return nil
			}
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"alert": cond.AlertID, "condition": cond.Condition}); err != nil {
				return err
			}
			// Route to the new API
			if cond.ConditionID != "" && cond.Channels != nil {
				err := config.Client.UpdateAlertSettingsCondition(strconv.Itoa(cond.AlertID), cond.Condition, cond.ConditionID, cond.Channels)
				if err != nil {
					return err
				}
				fmt.Fprintln(cmd.OutOrStdout(), "Update rule's channels succeeded")
				return nil
			}

			err := config.Client.CreateOrUpdateAlertSettingCondition(cond.AlertID, cond.Condition)
			if err != nil {
				golog.Errorf("Failed to creating/updating alert setting condition [%s]. [%s]", cond.Condition, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Condition [id=%d] added", cond.AlertID)
		},
	}

	cmd.Flags().IntVar(&cond.AlertID, "alert", 0, "Alert ID")
	cmd.Flags().StringVar(&cond.Condition, "condition", "", `Alert condition .e.g. "lag >= 100000 on group group and topic topicA"`)
	cmd.Flags().StringVar(&cond.ConditionID, "conditionID", "", "Alert condition ID")
	cmd.Flags().StringArrayVar(&cond.Channels, "channels", nil, "Channel UIDs")

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
				golog.Errorf("Failed to deleting alert setting condition [%s]. [%s]", conditionUUID, err.Error())
				return err
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

//NewGetAlertChannelsCommand creates the `alertchannels` command
func NewGetAlertChannelsCommand() *cobra.Command {
	var (
		page         int
		pageSize     int
		sortField    string
		sortOrder    string
		templateName string
		channelName  string
		details      bool
	)

	cmd := &cobra.Command{
		Use:              "alertchannels",
		Short:            "Print the registered alert channels",
		Example:          `alertchannels --page=1 --pageSize=10 --sortField="name" --sortOrder="asc" --templateName="test" --channelName="slack" --details`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if details {
				alertchannelsWithDetails, err := config.Client.GetAlertChannelsWithDetails(page, pageSize, sortField, sortOrder, templateName, channelName)
				if err != nil {
					golog.Errorf("Failed to retrieve alert channels. [%s]", err.Error())
					return err
				}
				return bite.PrintObject(cmd, alertchannelsWithDetails.Values)
			}

			alertchannels, err := config.Client.GetAlertChannels(page, pageSize, sortField, sortOrder, templateName, channelName)
			if err != nil {
				golog.Errorf("Failed to retrieve alert channels. [%s]", err.Error())
				return err
			}
			return bite.PrintObject(cmd, alertchannels.Values)
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "The page number to be fetched, must be greater than zero. Defaults to 1")
	cmd.Flags().IntVar(&pageSize, "pageSize", 10, "The amount of items to return in a single page, must be greater than zero.")
	cmd.Flags().StringVar(&sortField, "sortField", "", `The field to sort channel results by. Defaults to createdAt`)
	cmd.Flags().StringVar(&sortOrder, "sortOrder", "", `Choices: "asc" or "desc"`)
	cmd.Flags().StringVar(&templateName, "templateName", "", `Filter channels by template name.`)
	cmd.Flags().StringVar(&channelName, "channelName", "", `Filter channels whith a name matching the supplied string (e.g. kafka-prd would match kafka-prd-pagerduty and kafka-prd-slack).`)
	cmd.Flags().BoolVar(&details, "details", false, `--details`)

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)

	cmd.AddCommand(NewDeleteAlertChannelCommand())
	cmd.AddCommand(NewCreateAlertChannelCommand())
	cmd.AddCommand(NewUpdateAlertChannelCommand())

	return cmd
}

//NewDeleteAlertChannelCommand creates `alertchannels delete` command
func NewDeleteAlertChannelCommand() *cobra.Command {
	var (
		channelID string
	)

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an alert channel",
		Example:          `alertchannels delete --channelID="fa0e9b96-1048-4f4c-b776-4e96ca62f37d"`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := config.Client.DeleteAlertChannel(channelID)
			if err != nil {
				golog.Errorf("Failed to delete alert channel [%s]. [%s]", channelID, err.Error())
				return err
			}
			return bite.PrintInfo(cmd, "Alert channel [%s] deleted", channelID)
		},
	}

	cmd.Flags().StringVar(&channelID, "channelID", "", "The alert channel id, e.g. d15-4960-9ea6-2ccb4d26ebb4")
	cmd.MarkFlagRequired("channelID")
	bite.CanBeSilent(cmd)

	return cmd
}

// NewCreateAlertChannelCommand creates `alertchannels create` command
func NewCreateAlertChannelCommand() *cobra.Command {
	var (
		propertiesRaw string
		channel       = api.AlertChannelPayload{}
	)
	cmdExample := "\nalertchannels create --name=\"kafka-prd-health\" --templateName=\"Slack\" --connectionName=\"slack-connection\" --properties=\"[{\"key\":\"username\",\"value\":\"@luk\"},{\"key\":\"channel\",\"value\":\"#lenses\"}]\"\n" +
		"\n# or using YAML\nalertchannels create ./alert_chan.yml"

	cmd := &cobra.Command{
		Use:              "create",
		Short:            "Create a new alert channel",
		Example:          cmdExample,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": channel.Name, "connectionName": channel.ConnectionName, "templateName": channel.TemplateName, "properties": channel.Properties}); err != nil {
				return err
			}

			if propertiesRaw != "" {
				if err := bite.TryReadFile(propertiesRaw, &channel.Properties); err != nil {
					// from flag as json.
					if err = json.Unmarshal([]byte(propertiesRaw), &channel.Properties); err != nil {
						return fmt.Errorf("Unable to unmarshal the properties: [%v]", err)
					}
				}
			}

			if err := config.Client.CreateAlertChannel(channel); err != nil {
				golog.Errorf("Failed to create alert channel [%s]. [%s]", channel.Name, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Alert channel [%s] created", channel.Name)
		},
	}

	cmd.Flags().StringVar(&channel.Name, "name", "", "Alert channel name")
	cmd.Flags().StringVar(&channel.ConnectionName, "connectionName", "", "Alert channel connection name")
	cmd.Flags().StringVar(&channel.TemplateName, "templateName", "", "Alert channel template name")
	cmd.Flags().StringVar(&propertiesRaw, "properties", "", `Alert channel properties .e.g. "[{\"key\":\"username\",\"value\":\"@luk\"},{\"key\":\"channel\",\"value\":\"#lenses\"}]"`)

	bite.CanBeSilent(cmd)
	bite.Prepend(cmd, bite.FileBind(&channel))

	return cmd
}

// NewUpdateAlertChannelCommand creates `alertchannels create` command
func NewUpdateAlertChannelCommand() *cobra.Command {
	var (
		propertiesRaw string
		channelID     string
		channel       = api.AlertChannelPayload{}
	)

	cmd := &cobra.Command{
		Use:              "update",
		Short:            "Update an existing alert channel",
		Example:          `alertchannels update --id="fa0e9b96-1048-4f4c-b776-4e96ca62f37d" --name="kafka-prd-health" --templateName="Slack" --connectionName="slack-connection" --properties="[{\"key\":\"username\",\"value\":\"@luk\"},{\"key\":\"channel\",\"value\":\"#lenses\"}]"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"id": channelID, "name": channel.Name, "connectionName": channel.ConnectionName, "templateName": channel.TemplateName, "properties": channel.Properties}); err != nil {
				return err
			}

			if propertiesRaw != "" {
				if err := bite.TryReadFile(propertiesRaw, &channel.Properties); err != nil {
					// from flag as json.
					if err = json.Unmarshal([]byte(propertiesRaw), &channel.Properties); err != nil {
						return fmt.Errorf("Unable to unmarshal the properties: [%v]", err)
					}
				}
			}

			if err := config.Client.UpdateAlertChannel(channel, channelID); err != nil {
				golog.Errorf("Failed to update alert channel [%s]. [%s]", channelID, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Alert channel [%s] updated", channelID)
		},
	}

	cmd.Flags().StringVar(&channelID, "id", "", "The alert channel id, e.g. d15-4960-9ea6-2ccb4d26ebb4")
	cmd.Flags().StringVar(&channel.Name, "name", "", "Alert channel name")
	cmd.Flags().StringVar(&channel.ConnectionName, "connectionName", "", "Alert channel connection name")
	cmd.Flags().StringVar(&channel.TemplateName, "templateName", "", "Alert channel template name")
	cmd.Flags().StringVar(&propertiesRaw, "properties", "", `Alert channel properties .e.g. "[{\"key\":\"username\",\"value\":\"@luk\"},{\"key\":\"channel\",\"value\":\"#lenses\"}]"`)
	bite.CanBeSilent(cmd)
	bite.Prepend(cmd, bite.FileBind(&channel))

	return cmd
}
