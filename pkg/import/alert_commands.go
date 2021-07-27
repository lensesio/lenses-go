package imports

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

//NewImportAlertSettingsCommand create `import alert-settings` command
func NewImportAlertSettingsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "alert-settings",
		Short:            "alert-settings",
		Example:          `import alert-settings --dir <dir> --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.AlertSettingsPath)

			if err := loadProducerAlertSettings(config.Client, cmd, path); err != nil {
				return fmt.Errorf("failed to load alert-settings for data produced. [%s]", err.Error())
			}

			if err := loadConsumerAlertSettings(config.Client, cmd, path); err != nil {
				return fmt.Errorf("failed to load alert-settings for consumer rules. [%s]", err.Error())
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func loadConsumerAlertSettings(client *api.Client, cmd *cobra.Command, loadpath string) error {
	settings, err := client.GetAlertSetting(2000)
	if err != nil {
		return err
	}

	var existingConsumerAlertSettings api.ConsumerAlertSettings
	existingConsumerAlertSettings.ID = settings.ID
	existingConsumerAlertSettings.Description = settings.Description

	for _, condDetail := range settings.ConditionDetails {
		jsonStringCondition, _ := json.Marshal(condDetail.ConditionDsl)

		consumerAlertConditionDetail := api.ConsumerAlertConditionRequestv1{}
		json.Unmarshal(jsonStringCondition, &consumerAlertConditionDetail.Condition)

		for _, chann := range condDetail.Channels {
			consumerAlertConditionDetail.Channels = append(consumerAlertConditionDetail.Channels, chann.Name)
		}

		existingConsumerAlertSettings.ConditionDetails = append(existingConsumerAlertSettings.ConditionDetails, consumerAlertConditionDetail)
	}

	var targetConsumerAlertSettings api.ConsumerAlertSettings
	channels, err := client.GetChannels(pkg.AlertChannelsPath, 1, 99999, "", "", "", "")

	if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, "alert-setting-consumer.yaml"), &targetConsumerAlertSettings); err != nil {
		return fmt.Errorf("error loading file [%s]", loadpath)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "loading alert conditions from alert-setting-consumer.yaml\n")

	for _, targetCondition := range targetConsumerAlertSettings.ConditionDetails {
		var found bool

		for _, existingCondition := range existingConsumerAlertSettings.ConditionDetails {
			if existingCondition.Channels == nil {
				existingCondition.Channels = []string{}
			}

			if reflect.DeepEqual(targetCondition, existingCondition) {
				found = true
			}
		}
		if found {
			continue
		}

		targetConditionForLog := targetCondition
		targetConditionForLog.Channels = make([]string, len(targetCondition.Channels))
		copy(targetConditionForLog.Channels, targetCondition.Channels)

		for i, targetChann := range targetCondition.Channels {
			for _, chann := range channels.Values {
				if targetChann == chann.Name {
					targetCondition.Channels[i] = chann.ID
				}
			}
		}

		err := config.Client.SetAlertSettingsConsumerCondition(strconv.Itoa(2000), "",
			api.ConsumerAlertConditionRequestv1{Condition: targetCondition.Condition, Channels: targetCondition.Channels})

		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "created/updated condition: [%v]\n", targetConditionForLog)
	}

	return nil
}

func loadProducerAlertSettings(client *api.Client, cmd *cobra.Command, loadpath string) error {
	settings, err := client.GetAlertSetting(5000)
	if err != nil {
		return err
	}

	var existingProducerAlertSettings api.ProducerAlertSettings
	existingProducerAlertSettings.ID = settings.ID
	existingProducerAlertSettings.Description = settings.Description

	for _, condDetail := range settings.ConditionDetails {
		jsonStringCondition, _ := json.Marshal(condDetail.ConditionDsl)

		producerAlertConditionDetail := api.AlertConditionRequestv1{}
		json.Unmarshal(jsonStringCondition, &producerAlertConditionDetail.Condition)

		for _, chann := range condDetail.Channels {
			producerAlertConditionDetail.Channels = append(producerAlertConditionDetail.Channels, chann.Name)
		}

		existingProducerAlertSettings.ConditionDetails = append(existingProducerAlertSettings.ConditionDetails, producerAlertConditionDetail)
	}

	var targetProducerAlertSettings api.ProducerAlertSettings
	channels, err := client.GetChannels(pkg.AlertChannelsPath, 1, 99999, "", "", "", "")

	if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, "alert-setting-producer.yaml"), &targetProducerAlertSettings); err != nil {
		return fmt.Errorf("error loading file [%s]", loadpath)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "loading alert conditions from alert-setting-producer.yaml\n")

	for _, targetCondition := range targetProducerAlertSettings.ConditionDetails {
		var found bool

		for _, existingCondition := range existingProducerAlertSettings.ConditionDetails {
			if existingCondition.Channels == nil {
				existingCondition.Channels = []string{}
			}

			if reflect.DeepEqual(targetCondition, existingCondition) {
				found = true
			}
		}
		if found {
			continue
		}

		targetConditionForLog := targetCondition
		targetConditionForLog.Channels = make([]string, len(targetCondition.Channels))
		copy(targetConditionForLog.Channels, targetCondition.Channels)

		for i, targetChann := range targetCondition.Channels {
			for _, chann := range channels.Values {
				if targetChann == chann.Name {
					targetCondition.Channels[i] = chann.ID
				}
			}
		}
		err := config.Client.SetAlertSettingsProducerCondition(
			strconv.Itoa(5000), "",
			targetCondition.Condition.DatasetName,
			targetCondition.Condition.Threshold,
			targetCondition.Condition.Duration,
			targetCondition.Channels)

		if err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "created/updated condition: [%v]\n", targetConditionForLog)
	}

	return nil
}
