package audit

import (
	"fmt"

	"encoding/json"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

//NewGetAuditChannelsCommand creates the `auditchannels` command
func NewGetAuditChannelsCommand() *cobra.Command {
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
		Use:              "auditchannels",
		Short:            "Print the registered audit channels",
		Example:          `auditchannels --page=1 --pageSize=10 --sortField="name" --sortOrder="asc" --templateName="test" --channelName="slack" --details`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			auditChannelsPath := pkg.AuditChannelsPath

			if details {
				auditchannelsWithDetails, err := config.Client.GetChannelsWithDetails(auditChannelsPath, page, pageSize, sortField, sortOrder, templateName, channelName)
				if err != nil {
					return fmt.Errorf("failed to retrieve audits' channels. Error: [%s]", err.Error())
				}
				return bite.PrintObject(cmd, auditchannelsWithDetails.Values)
			}

			auditchannels, err := config.Client.GetChannels(auditChannelsPath, page, pageSize, sortField, sortOrder, templateName, channelName)
			if err != nil {
				return fmt.Errorf("failed to retrieve audits' channels. Error: [%s]", err.Error())
			}
			return bite.PrintObject(cmd, auditchannels.Values)
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

	cmd.AddCommand(NewDeleteAuditChannelCommand())
	cmd.AddCommand(NewCreateAuditChannelCommand())
	cmd.AddCommand(NewUpdateAuditChannelCommand())

	return cmd
}

//NewDeleteAuditChannelCommand creates `auditchannels delete` command
func NewDeleteAuditChannelCommand() *cobra.Command {
	var (
		channelID string
	)

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an audit channel",
		Example:          `auditchannels delete --channelID="fa0e9b96-1048-4f4c-b776-4e96ca62f37d"`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := config.Client.DeleteChannel(pkg.AuditChannelsPath, channelID)
			if err != nil {
				return fmt.Errorf("failed to delete audit channel [%s]. [%s]", channelID, err.Error())
			}
			return bite.PrintInfo(cmd, "Audit channel [%s] deleted", channelID)
		},
	}

	cmd.Flags().StringVar(&channelID, "channelID", "", "The audit channel id, e.g. d15-4960-9ea6-2ccb4d26ebb4")
	cmd.MarkFlagRequired("channelID")
	bite.CanBeSilent(cmd)

	return cmd
}

// NewCreateAuditChannelCommand creates `auditchannels create` command
func NewCreateAuditChannelCommand() *cobra.Command {
	var (
		propertiesRaw string
		channel       = api.ChannelPayload{}
	)
	cmdExample := "\nauditchannels create --name=\"kafka-prd-health\" --templateName=\"Slack\" --connectionName=\"slack-connection\" --properties=\"[{\"key\":\"username\",\"value\":\"@luk\"},{\"key\":\"channel\",\"value\":\"#lenses\"}]\"\n" +
		"\n# or using YAML\nauditchannels create ./audit_chan.yml"

	cmd := &cobra.Command{
		Use:              "create",
		Short:            "Create a new audit channel",
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

			if err := config.Client.CreateChannel(channel, pkg.AuditChannelsPath); err != nil {
				return fmt.Errorf("failed to create audit channel [%s]. [%s]", channel.Name, err.Error())
			}

			return bite.PrintInfo(cmd, "audit channel [%s] created", channel.Name)
		},
	}

	cmd.Flags().StringVar(&channel.Name, "name", "", "Audit channel name")
	cmd.Flags().StringVar(&channel.ConnectionName, "connectionName", "", "Audit channel connection name")
	cmd.Flags().StringVar(&channel.TemplateName, "templateName", "", "Audit channel template name")
	cmd.Flags().StringVar(&propertiesRaw, "properties", "", `Audit channel properties .e.g. "[{\"key\":\"username\",\"value\":\"@luk\"},{\"key\":\"channel\",\"value\":\"#lenses\"}]"`)

	bite.CanBeSilent(cmd)
	bite.Prepend(cmd, bite.FileBind(&channel))

	return cmd
}

// NewUpdateAuditChannelCommand creates `auditchannels create` command
func NewUpdateAuditChannelCommand() *cobra.Command {
	var (
		propertiesRaw string
		channelID     string
		channel       = api.ChannelPayload{}
	)

	cmd := &cobra.Command{
		Use:              "update",
		Short:            "Update an existing audit channel",
		Example:          `auditchannels update --id="fa0e9b96-1048-4f4c-b776-4e96ca62f37d" --name="kafka-prd-health" --templateName="Slack" --connectionName="slack-connection" --properties="[{\"key\":\"username\",\"value\":\"@luk\"},{\"key\":\"channel\",\"value\":\"#lenses\"}]"`,
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

			if err := config.Client.UpdateChannel(channel, pkg.AuditChannelsPath, channelID); err != nil {
				return fmt.Errorf("failed to update audit channel [%s]. [%s]", channelID, err.Error())
			}

			return bite.PrintInfo(cmd, "audit channel [%s] updated", channelID)
		},
	}

	cmd.Flags().StringVar(&channelID, "id", "", "The audit channel id, e.g. d15-4960-9ea6-2ccb4d26ebb4")
	cmd.Flags().StringVar(&channel.Name, "name", "", "Audit channel name")
	cmd.Flags().StringVar(&channel.ConnectionName, "connectionName", "", "Audit channel connection name")
	cmd.Flags().StringVar(&channel.TemplateName, "templateName", "", "Audit channel template name")
	cmd.Flags().StringVar(&propertiesRaw, "properties", "", `Audit channel properties .e.g. "[{\"key\":\"username\",\"value\":\"@luk\"},{\"key\":\"channel\",\"value\":\"#lenses\"}]"`)
	bite.CanBeSilent(cmd)
	bite.Prepend(cmd, bite.FileBind(&channel))

	return cmd
}
