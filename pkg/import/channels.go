package imports

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

func importChannels(client *api.Client, cmd *cobra.Command, loadpath, channelType, channelsPath string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "loading %s channels from [%s] directory\n", channelType, loadpath)

	var targetChannels []api.ChannelPayload
	files, err := utils.FindFiles(loadpath)
	if err != nil {
		return err
	}

	for _, file := range files {
		var targetChannel api.ChannelPayload
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &targetChannel); err != nil {
			return fmt.Errorf("error loading file [%s]", loadpath)
		}
		targetChannels = append(targetChannels, targetChannel)
	}

	channels, err := client.GetChannels(channelsPath, 1, 99999, "name", "asc", "", "")
	if err != nil {
		return err
	}

	var sourceChannels []api.ChannelPayload
	for _, chann := range channels.Values {
		var channForExport api.ChannelPayload
		channΑsJSON, _ := json.Marshal(chann)
		json.Unmarshal(channΑsJSON, &channForExport)
		sourceChannels = append(sourceChannels, channForExport)
	}

	// Check for duplicates lacking server-side implementation
	for _, targetChannel := range targetChannels {
		found := false

		for _, sourceChannel := range sourceChannels {
			if reflect.DeepEqual(targetChannel, sourceChannel) {
				found = true
			}
		}

		if found {
			continue
		}

		if err := client.CreateChannel(targetChannel, channelsPath); err != nil {
			return fmt.Errorf("error importing %s channel [%v]", channelType, targetChannel)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s channel [%s] successfully imported\n", channelType, targetChannel.Name)
	}

	return nil
}
