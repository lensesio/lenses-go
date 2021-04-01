package export

import (
	"fmt"
	"strings"

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

// NewExportTopicSettingsCmd to export topic-settings to yaml
func NewExportTopicSettingsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use: "topic-settings",
		Long: heredoc.Doc(`
			Export Topic Settings

			The settings can be exported to different Kafka Cluster to allow for best GitOps practices.
		`),
		Example: heredoc.Doc(`
			$ lenses-cli export
		`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			setExecutionMode(client)
			checkFileFlags(cmd)

			err := WriteTopicSettings(cmd, client)
			return errors.WithStack(err)

		},
	}

	cmd.Flags().StringVarP(&landscapeDir, "dir", "D", ".", "Base directory to export to")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)

	return cmd
}

// WriteTopicSettings to a file
func WriteTopicSettings(cmd *cobra.Command, client *api.Client) error {
	golog.Infof("Writing topic-settings to [%s]", landscapeDir)

	settings, err := client.GetTopicSettings()

	if err != nil {
		return errors.WithStack(err)
	}

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("topic-settings.%s", strings.ToLower(output))
	return utils.WriteFile(landscapeDir, pkg.TopicSettingsPath, fileName, output, settings)
}
