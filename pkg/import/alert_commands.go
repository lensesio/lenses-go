package imports

import (
	"fmt"

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
		Example:          `import alert-settings --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.AlertSettingsPath)
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
	golog.Infof("Loading alert-settings from [%s]", loadpath)
	files := utils.FindFiles(loadpath)

	asc, err := client.GetAlertSettingConditions(2000)

	if err != nil {
		return err
	}

	for _, file := range files {

		var conds alert.SettingConditionPayloads
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &conds); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

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

			if err := client.CreateOrUpdateAlertSettingCondition(alertID, condition); err != nil {
				golog.Errorf("Error creating/updating alert setting from [%d] [%s] [%s]", alertID, loadpath, err.Error())
				return err
			}
			golog.Infof("Created/updated condition [%s] from [%s]", condition, loadpath)
		}
	}
	return nil
}
