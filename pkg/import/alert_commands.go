package imports

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/alert"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
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

			if err := loadAlertSettings(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to load alert-settings. [%s]", err.Error())
				return err
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

func loadAlertSettings(client *api.Client, cmd *cobra.Command, loadpath string) error {

	asc, err := client.GetAlertSettingConditions(2000)

	if err != nil {
		return err
	}

	var conds alert.SettingConditionPayloads

	if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, "alert-setting.yaml"), &conds); err != nil {
		return fmt.Errorf("error loading file [%s]", loadpath)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "loading alert conditions from alert-setting.yaml\n")
	alertID := conds.AlertID

	for _, condition := range conds.Conditions {
		found := false
		for _, v := range asc {
			if v == condition {
				found = true
			}
		}

		if found {
			continue
		}

		if err := client.CreateAlertSettingsCondition(strconv.Itoa(alertID), condition, []string{}); err != nil {
			return fmt.Errorf("error creating/updating alert setting from from [%d] [%s] [%s]", alertID, loadpath, err.Error())
		}
		fmt.Fprintf(cmd.OutOrStdout(), "created/updated condition [%s]\n", condition)
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
	channels, err := client.GetAlertChannels(1, 99999, "", "", "", "")

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

//NewImportAlertChannelsCommand handles the CLI sub-command 'import alert-channels'
func NewImportAlertChannelsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "alert-channels",
		Short:            "alert-channels",
		Example:          `import alert-channels --dir <dir>`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, "alert-channels")

			err := importAlertChannels(config.Client, cmd, path)
			if err != nil {
				return fmt.Errorf("error importing alert channels. [%v]", err)
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

func importAlertChannels(client *api.Client, cmd *cobra.Command, loadpath string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "loading alert channels from [%s] directory\n", loadpath)

	var targetAlertChannels []api.AlertChannelPayload
	files := utils.FindFiles(loadpath)
	for _, file := range files {
		var targetAlertChannel api.AlertChannelPayload
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &targetAlertChannel); err != nil {
			return fmt.Errorf("error loading file [%s]", loadpath)
		}
		targetAlertChannels = append(targetAlertChannels, targetAlertChannel)
	}

	channels, err := client.GetAlertChannels(1, 99999, "name", "asc", "", "")
	if err != nil {
		return err
	}

	var sourceAlertChannels []api.AlertChannelPayload
	for _, chann := range channels.Values {
		var channForExport api.AlertChannelPayload
		channΑsJSON, _ := json.Marshal(chann)
		json.Unmarshal(channΑsJSON, &channForExport)
		sourceAlertChannels = append(sourceAlertChannels, channForExport)
	}

	// Check for duplicates lacking server-side implementation
	for _, targetChannel := range targetAlertChannels {
		found := false

		for _, sourceChannel := range sourceAlertChannels {
			if reflect.DeepEqual(targetChannel, sourceChannel) {
				found = true
			}
		}

		if found {
			continue
		}

		if err := client.CreateAlertChannel(targetChannel); err != nil {
			return fmt.Errorf("error importing alert channel [%v]", targetChannel)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "alert channel [%s] successfully imported\n", targetChannel.Name)
	}

	return nil
}
