package imports

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/kataras/golog"
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
			return errors.Wrap(err, "Failed to read topic-settings")
		},
	}

	cmd.Flags().StringVarP(&path, "dir", "D", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")

	return cmd
}

// ReadTopicSettings to read for each file and pass the topic-settings
func ReadTopicSettings(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Reading topic-settings from [%s]", loadpath)

	files := utils.FindFiles(loadpath)

	for _, file := range files {
		var settings api.TopicSettingsRequest
		fileErr := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &settings)

		if fileErr != nil {
			return errors.WithStack(fileErr)
		}

		updateErr := client.UpdateTopicSettings(settings)
		fmt.Println(utils.GREEN("✓ Imported Topic Settings"))

		if fileErr != nil {
			return errors.WithStack(updateErr)
		}
	}

	return nil
}
