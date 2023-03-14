package alert

import (
	"encoding/json"
	"fmt"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/spf13/cobra"
)

// NewGetAlertChannelsCommand creates the `alertchannels` command
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
				alertchannelsWithDetails, err := config.Client.GetChannelsWithDetails(pkg.AlertChannelsPath, page, pageSize, sortField, sortOrder, templateName, channelName)
				if err != nil {
					return fmt.Errorf("failed to retrieve alerts' channels. Error: [%s]", err.Error())
				}
				return bite.PrintObject(cmd, alertchannelsWithDetails.Values)
			}

			alertchannels, err := config.Client.GetChannels(pkg.AlertChannelsPath, page, pageSize, sortField, sortOrder, templateName, channelName)
			if err != nil {
				return fmt.Errorf("failed to retrieve alerts' channels. Error: [%s]", err.Error())
			}
			return bite.PrintObject(cmd, alertchannels.Values)
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "The page number to be fetched, must be greater than zero. Defaults to 1")
	cmd.Flags().IntVar(&pageSize, "pageSize", 10, "The amount of items to return in a single page, must be greater than zero.")
	cmd.Flags().StringVar(&sortField, "sortField", "", `The field to sort channel results by. Defaults to createdAt`)
	cmd.Flags().StringVar(&sortOrder, "sortOrder", "", `Choices: "asc" or "desc"`)
	cmd.Flags().StringVar(&templateName, "templateName", "", `Filter channels by template name.`)
	cmd.Flags().StringVar(&channelName, "channelName", "", `Filter channels with a name matching the supplied string (e.g. kafka-prd would match kafka-prd-pagerduty and kafka-prd-slack).`)
	cmd.Flags().BoolVar(&details, "details", false, `--details`)

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)

	cmd.AddCommand(NewDeleteAlertChannelCommand())
	cmd.AddCommand(NewCreateAlertChannelCommand())
	cmd.AddCommand(NewUpdateAlertChannelCommand())

	return cmd
}

// NewDeleteAlertChannelCommand creates `alertchannels delete` command
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
			err := config.Client.DeleteChannel(pkg.AlertChannelsPath, channelID)
			if err != nil {
				return fmt.Errorf("failed to delete alert channel [%s]. [%s]", channelID, err.Error())
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
		channel       = api.ChannelPayload{}
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
						return fmt.Errorf("unable to unmarshal the properties: [%v]", err)
					}
				}
			}

			if err := config.Client.CreateChannel(channel, pkg.AlertChannelsPath); err != nil {
				return fmt.Errorf("failed to create alert channel [%s]. [%s]", channel.Name, err.Error())
			}

			return bite.PrintInfo(cmd, "alert channel [%s] created", channel.Name)
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
		channel       = api.ChannelPayload{}
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
						return fmt.Errorf("unable to unmarshal the properties: [%v]", err)
					}
				}
			}

			if err := config.Client.UpdateChannel(channel, pkg.AlertChannelsPath, channelID); err != nil {
				return fmt.Errorf("failed to update alert channel [%s]. [%s]", channelID, err.Error())
			}

			return bite.PrintInfo(cmd, "alert channel [%s] updated", channelID)
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
