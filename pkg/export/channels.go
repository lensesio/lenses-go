package export

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

func writeChannels(cmd *cobra.Command, channelsPath, channelType, channelName string) error {
	channels, err := config.Client.GetChannels(channelsPath, 1, 99999, "name", "asc", "", "")
	if err != nil {
		return fmt.Errorf("failed to retrieve channels from server: [%v]", err)
	}

	if channelName != "" {
		for _, channel := range channels.Values {
			if channelName == channel.Name {
				// fileName := fmt.Sprintf("%s-channel-%s.%s", channelType, strings.ToLower(channel.Name), strings.ToLower(bite.GetOutPutFlag(cmd)))
				// subDir := channelType+"-channels"

				// utils.WriteFile(landscapeDir, subDir, fileName, strings.ToUpper(bite.GetOutPutFlag(cmd)), channel)
				// fmt.Fprintf(cmd.OutOrStdout(), "exporting [%s] %s channel to base directory [%s]\n", channelName, channelType, landscapeDir)

				// return nil

				writeChannelToFile(cmd, channelType, channel.Name, channel)
			}
		}

		return fmt.Errorf("%s channel with name [%s] was not found", channelType, channelName)
	}

	var channelsForExport []api.ChannelPayload
	for _, chann := range channels.Values {
		var channForExport api.ChannelPayload
		channΑsJSON, _ := json.Marshal(chann)
		json.Unmarshal(channΑsJSON, &channForExport)
		channelsForExport = append(channelsForExport, channForExport)
	}

	for _, channelForExport := range channelsForExport {
		writeChannelToFile(cmd, channelType, channelForExport.Name, channelForExport)
	}

	return nil
}

func writeChannelToFile(cmd *cobra.Command, channelType, channelName string, channel interface{}) error {
	fileName := fmt.Sprintf("%s-channel-%s.%s", channelType, strings.ToLower(channelName), strings.ToLower(bite.GetOutPutFlag(cmd)))
	subDir := channelType + "-channels"

	utils.WriteFile(landscapeDir, subDir, fileName, strings.ToUpper(bite.GetOutPutFlag(cmd)), channel)

	fmt.Fprintf(cmd.OutOrStdout(), "exported %s channel [%s] to [%s]\n", channelType, channelName, fileName)

	return nil
}
