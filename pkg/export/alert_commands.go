package export

import (
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

func writeAlertSetting(cmd *cobra.Command, client alert.Client) error {
	producerSettings, err := alert.GetProducerAlertSettings(client)
	if err != nil {
		return err
	}
	consumerSettings, err := alert.GetConsumerAlertSettings(client)
	if err != nil {
		return err
	}
	if err := writeProducerAlertSettingsV1(cmd, producerSettings); err != nil {
		return fmt.Errorf("write producer settings: %w", err)
	}
	if err := writeConsumerAlertSettingsV1(cmd, consumerSettings); err != nil {
		return fmt.Errorf("write consumer settings: %w", err)
	}
	return nil
}

func writeProducerAlertSettingsV1(cmd *cobra.Command, settings api.ProducerAlertSettings) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("alert-setting-producer.%s", strings.ToLower(output))

	err := utils.WriteFile(landscapeDir, pkg.AlertSettingsPath, fileName, output, settings)
	if err != nil {
		return fmt.Errorf("error writing to %s. [%v]", fileName, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "successfully wrote to %s\n", fileName)
	return nil
}

func writeConsumerAlertSettingsV1(cmd *cobra.Command, settings api.ConsumerAlertSettings) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("alert-setting-consumer.%s", strings.ToLower(output))

	err := utils.WriteFile(landscapeDir, pkg.AlertSettingsPath, fileName, output, settings)
	if err != nil {
		return fmt.Errorf("error writing to %s. [%v]", fileName, err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "successfully wrote to %s\n", fileName)
	return nil
}

// getConsumerAlertsByTopics  goes through the alerts of category "consumer". If
// the "conditions" expression contains any of the provided topics the condition
// is added to the return value. It gives control to the v2 alert settings
// endpoint if available, or falls back to the v1 otherwise. It is used in the
// "dependents" logic.
func getConsumerAlertsByTopics(cmd *cobra.Command, client *api.Client, topics []string) (alert.SettingConditionPayloads, error) {
	useV2, err := client.HasAlertSettingsV2Endpoints()
	if err != nil {
		return alert.SettingConditionPayloads{}, fmt.Errorf("has alert settings v2 endpoints: %w", err)
	}
	if useV2 {
		return getConsumerAlertsByTopicsV2(cmd, client, topics)
	}
	return getConsumerAlertsByTopicsV1(cmd, client, topics)
}

// getConsumerAlertsByTopicsV1 goes through the alerts of category "consumer".
// If the "conditions" expression contains any of the provided topics the
// condition is added to the return value.
func getConsumerAlertsByTopicsV1(cmd *cobra.Command, client *api.Client, topics []string) (alert.SettingConditionPayloads, error) {
	settings, err := client.GetAlertSettingsV1()
	if err != nil {
		return alert.SettingConditionPayloads{}, err
	}

	if len(settings.Categories.Consumers) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "no alert settings found")
		return alert.SettingConditionPayloads{}, nil
	}

	var conditions []string
	for _, setting := range settings.Categories.Consumers {
		for _, condition := range setting.Conditions {
			for _, topic := range topics {
				// A condition is e.g.: "lag >= 42 on group my-consumer-group and topic my-topic".
				if strings.Contains(condition, "topic "+topic) {
					conditions = append(conditions, condition)
				}
			}
		}
	}

	if len(conditions) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "no consumer conditions found\n")
		return alert.SettingConditionPayloads{}, nil
	}

	return alert.SettingConditionPayloads{AlertID: 2000, Conditions: conditions}, nil
}

// getConsumerAlertsByTopics goes through the alerts of category "consumer". If
// the "conditions" expression contains any of the provided topics the condition
// is added to the return value. Its behaviour aims to replicate that of the
// original v1 version [getConsumerAlertsByTopicsV1].
func getConsumerAlertsByTopicsV2(cmd *cobra.Command, client *api.Client, topics []string) (alert.SettingConditionPayloads, error) {
	rule, err := client.GetAlertSettingV2(2000)
	if err != nil {
		return alert.SettingConditionPayloads{}, err
	}

	if rule.Details.AlertType != api.AlertTypeConditional {
		return alert.SettingConditionPayloads{}, fmt.Errorf("expected a conditional alert type, got: %q", rule.Details.AlertType)
	}

	var conditions []string
	for _, condition := range rule.Details.Conditional.Conditions {
		for _, topic := range topics {
			// A condition is e.g.: "lag >= 42 on group my-consumer-group and topic my-topic".
			if strings.Contains(condition.Condition, "topic "+topic) {
				conditions = append(conditions, condition.Condition)
			}
		}
	}

	if len(conditions) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "no consumer conditions found\n")
		return alert.SettingConditionPayloads{}, nil
	}

	return alert.SettingConditionPayloads{AlertID: 2000, Conditions: conditions}, nil
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
