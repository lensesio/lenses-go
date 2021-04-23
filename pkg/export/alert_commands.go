package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/alert"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewExportAlertsCommand creates `export alert-settings` command
func NewExportAlertsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "alert-settings",
		Short:            "export alert-settings",
		Example:          `export alert-settings --resource-name=my-alert`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			if err := writeAlertSetting(cmd, config.Client); err != nil {
				return fmt.Errorf("error writing alert settings. [%s]", err.Error())
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract dependencies, topics, acls, quotas, alerts")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeAlertSetting(cmd *cobra.Command, client *api.Client) error {

	producerSettings, err := getProducerAlertSettings(client)
	if err != nil {
		return err
	}
	consumerSettings, err := getConsumerAlertSettings(client)
	if err != nil {
		return err
	}
	writeProducerAlertSettings(cmd, producerSettings)
	writeConsumerAlertSettings(cmd, consumerSettings)

	return nil
}

func writeProducerAlertSettings(cmd *cobra.Command, settings api.ProducerAlertSettings) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("alert-setting-producer.%s", strings.ToLower(output))

	err := utils.WriteFile(landscapeDir, pkg.AlertSettingsPath, fileName, output, settings)
	if err != nil {
		return fmt.Errorf("error writing to %s. [%v]", fileName, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "successfully wrote to %s\n", fileName)
	return nil
}

func writeConsumerAlertSettings(cmd *cobra.Command, settings api.ConsumerAlertSettings) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("alert-setting-consumer.%s", strings.ToLower(output))

	err := utils.WriteFile(landscapeDir, pkg.AlertSettingsPath, fileName, output, settings)
	if err != nil {
		return fmt.Errorf("error writing to %s. [%v]", fileName, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "successfully wrote to %s\n", fileName)
	return nil
}

func writeAlertSettingsAsRequest(cmd *cobra.Command, settings alert.SettingConditionPayloads) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("alert-setting.%s", strings.ToLower(output))

	err := utils.WriteFile(landscapeDir, pkg.AlertSettingsPath, fileName, output, settings)
	if err != nil {
		return fmt.Errorf("error writing to %s. [%v]", fileName, err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "successfully wrote to %s\n", fileName)
	return nil
}

func getAlertSettings(cmd *cobra.Command, client *api.Client, topics []string) (alert.SettingConditionPayloads, error) {
	var alertSettings alert.SettingConditionPayloads
	var conditions []string

	settings, err := client.GetAlertSettings()

	if err != nil {
		return alertSettings, err
	}

	if len(settings.Categories.Consumers) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "no alert settings found")
		return alertSettings, nil
	}

	consumerSettings := settings.Categories.Consumers

	for _, setting := range consumerSettings {
		for _, condition := range setting.Conditions {
			if len(topics) == 0 {
				conditions = append(conditions, condition)
				continue
			}

			// filter by topic name
			for _, topic := range topics {
				if strings.Contains(condition, fmt.Sprintf("topic %s", topic)) {
					conditions = append(conditions, condition)
				}
			}
		}
	}

	if len(conditions) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "no consumer conditions found\n")
		return alertSettings, nil
	}

	return alert.SettingConditionPayloads{AlertID: 2000, Conditions: conditions}, nil
}

func getConsumerAlertSettings(client *api.Client) (api.ConsumerAlertSettings, error) {
	var consumerAlertSettings api.ConsumerAlertSettings

	settings, err := client.GetAlertSetting(2000)
	if err != nil {
		return consumerAlertSettings, err
	}

	consumerAlertSettings.ID = settings.ID
	consumerAlertSettings.Description = settings.Description

	// iterate over the consumer condition details
	for _, condDetail := range settings.ConditionDetails {
		jsonStringCondition, _ := json.Marshal(condDetail.ConditionDsl)

		consumerAlertConditionDetail := api.ConsumerAlertConditionRequestv1{}
		json.Unmarshal(jsonStringCondition, &consumerAlertConditionDetail.Condition)

		// iterate channels of a condition detail
		for _, chann := range condDetail.Channels {
			consumerAlertConditionDetail.Channels = append(consumerAlertConditionDetail.Channels, chann.Name)
		}

		consumerAlertSettings.ConditionDetails = append(consumerAlertSettings.ConditionDetails, consumerAlertConditionDetail)
	}

	return consumerAlertSettings, nil
}

func getProducerAlertSettings(client *api.Client) (api.ProducerAlertSettings, error) {
	var producerAlertSettings api.ProducerAlertSettings

	settings, err := client.GetAlertSetting(5000)
	if err != nil {
		return producerAlertSettings, err
	}

	producerAlertSettings.ID = settings.ID
	producerAlertSettings.Description = settings.Description

	// iterate over the data produced condition details
	for _, condDetail := range settings.ConditionDetails {
		jsonStringCondition, _ := json.Marshal(condDetail.ConditionDsl)

		producerAlertConditionDetail := api.AlertConditionRequestv1{}
		json.Unmarshal(jsonStringCondition, &producerAlertConditionDetail.Condition)

		// iterate channels of a condition detail
		for _, chann := range condDetail.Channels {
			producerAlertConditionDetail.Channels = append(producerAlertConditionDetail.Channels, chann.Name)
		}

		producerAlertSettings.ConditionDetails = append(producerAlertSettings.ConditionDetails, producerAlertConditionDetail)
	}

	return producerAlertSettings, nil
}

//NewExportAlertChannelsCommand creates `export alert-channels` command
func NewExportAlertChannelsCommand() *cobra.Command {
	var alertChannelName string

	cmd := &cobra.Command{
		Use:              "alert-channels",
		Short:            "export alert-channels",
		Example:          `export alert-channels --resource-name=my-alert`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			if err := writeAlertChannels(cmd, alertChannelName); err != nil {
				return fmt.Errorf("failed to export alert channels from server: [%v]", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().StringVar(&alertChannelName, "resource-name", "", "The name of the alert channel to export")
	//cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract alert channel dependencies, e.g. connections")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeAlertChannels(cmd *cobra.Command, channelName string) error {
	channels, err := config.Client.GetChannels(pkg.AlertChannelsPath, 1, 99999, "name", "asc", "", "")
	if err != nil {
		return fmt.Errorf("failed to retrieve alert channels from server: [%v]", err)
	}

	if channelName != "" {
		for _, channel := range channels.Values {
			if channelName == channel.Name {
				fileName := fmt.Sprintf("alert-channel-%s.%s", strings.ToLower(channel.Name), strings.ToLower(bite.GetOutPutFlag(cmd)))
				utils.WriteFile(landscapeDir, "alert-channels", fileName, strings.ToUpper(bite.GetOutPutFlag(cmd)), channel)
				fmt.Fprintf(cmd.OutOrStdout(), "exporting [%s] alert channel to base directory [%s]\n", channelName, landscapeDir)

				return nil
			}
		}

		return fmt.Errorf("alert channel with name [%s] was not found", channelName)
	}

	var channelsForExport []api.AlertChannelPayload
	for _, chann := range channels.Values {
		var channForExport api.AlertChannelPayload
		channΑsJSON, _ := json.Marshal(chann)
		json.Unmarshal(channΑsJSON, &channForExport)
		channelsForExport = append(channelsForExport, channForExport)
	}

	for _, channelForExport := range channelsForExport {
		fileName := fmt.Sprintf("alert-channel-%s.%s", strings.ToLower(channelForExport.Name), strings.ToLower(bite.GetOutPutFlag(cmd)))
		utils.WriteFile(landscapeDir, "alert-channels", fileName, strings.ToUpper(bite.GetOutPutFlag(cmd)), channelForExport)
		fmt.Fprintf(cmd.OutOrStdout(), "exported alert channel [%s] to [%s]\n", channelForExport.Name, fileName)
	}

	return nil
}
