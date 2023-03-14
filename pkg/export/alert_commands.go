package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/alert"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportAlertsCommand creates `export alert-settings` command
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
