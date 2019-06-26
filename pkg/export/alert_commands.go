package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg"
	"github.com/landoop/lenses-go/pkg/alert"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/utils"
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
				golog.Errorf("Error writing alert-settings. [%s]", err.Error())
				return err
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

	var topics []string
	settings, err := getAlertSettings(cmd, client, topics)

	if err != nil {
		return err
	}

	writeAlertSettingsAsRequest(cmd, settings)

	return nil
}

func writeAlertSettingsAsRequest(cmd *cobra.Command, settings alert.SettingConditionPayloads) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("alert-setting.%s", strings.ToLower(output))

	return utils.WriteFile(landscapeDir, pkg.AlertSettingsPath, fileName, output, settings)
}

func getAlertSettings(cmd *cobra.Command, client *api.Client, topics []string) (alert.SettingConditionPayloads, error) {
	var alertSettings alert.SettingConditionPayloads
	var conditions []string

	settings, err := client.GetAlertSettings()

	if err != nil {
		return alertSettings, err
	}

	if len(settings.Categories.Consumers) == 0 {
		bite.PrintInfo(cmd, "No alert settings found ")
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
		bite.PrintInfo(cmd, "No consumer conditions found ")
		return alertSettings, nil
	}

	return alert.SettingConditionPayloads{AlertID: 2000, Conditions: conditions}, nil
}
