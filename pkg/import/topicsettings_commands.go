package imports

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ImportTopicSettingsCmd to read topic-settings from files
func ImportTopicSettingsCmd() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use: "topic-settings",
		Long: heredoc.Doc(`
			Import Topic Settings

			The settings can be import from a different Kafka Cluster to allow for best GitOps practices.
		`),
		Example: heredoc.Doc(`
			$ lenses-cli import topic-settings --dir /directory
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path = fmt.Sprintf("%s/%s", path, pkg.TopicSettingsPath)
			err := ReadTopicSettings(config.Client, cmd, path)
			return errors.Wrapf(err, utils.RED("Failed to read topic-settings"))
		},
	}

	cmd.Flags().StringVarP(&path, "dir", "D", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")

	return cmd
}

// ReadTopicSettings to read for each file and pass the topic-settings
func ReadTopicSettings(client *api.Client, cmd *cobra.Command, filePath string) error {
	files, err := utils.FindFiles(filePath)
	if err != nil {
		return err
	}

	for _, file := range files {
		var settings api.TopicSettingsRequest
		var fileName = file.Name()
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", filePath, file.Name()), &settings); err != nil {
			return errors.Wrapf(err, utils.RED("Could not load file [%s]"), fileName)
		}

		if err := client.UpdateTopicSettings(settings); err != nil {
			return errors.Wrapf(err, utils.RED("Could not update Topic Settings [%s]"), fileName)
		}
		fmt.Printf(utils.GREEN("✓ Imported Topic Settings from [%s]\n"), fileName)
	}

	return nil
}
