package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

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
