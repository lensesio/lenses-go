package imports

import (
	"fmt"

	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/utils"

	"github.com/kataras/golog"
	"github.com/spf13/cobra"
)

var importDir string

//NewImportProcessorsCommand import processors command
func NewImportProcessorsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "processors",
		Short:            "processors",
		Example:          `import processors --dir /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, pkg.SQLPath)
			if err := loadProcessors(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to load processors. [%s]", err.Error())
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

func loadProcessors(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading processors from [%s]", loadpath)
	files := utils.FindFiles(loadpath)

	processors, err := client.GetProcessors()

	if err != nil {
		golog.Errorf("Failed to retrieve processors. [%s]", err.Error())
	}

	for _, file := range files {

		var processor api.CreateProcessorPayload

		if err := load(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &processor); err != nil {
			return err
		}

		for _, p := range processors {
			if processor.Name == p.Name &&
				processor.ClusterName == p.ClusterName &&
				processor.Namespace == p.Namespace {

				if processor.Runners != p.Runners {
					//scale
					if err := client.UpdateProcessorRunners(p.ID, processor.Runners); err != nil {
						golog.Errorf("Error scaling processor [%s] from file [%s/%s]. [%s]", p.ID, loadpath, file.Name(), err.Error())
						return err
					}
					golog.Infof("Scaled processor [%s] from file [%s/%s] from [%d] to [%d]", p.ID, loadpath, file.Name(), p.Runners, processor.Runners)
					return nil
				}
				golog.Warnf("Processor [%s] from file [%s/%s] already exists", p.ID, loadpath, file.Name())
			}
		}

		if err := client.CreateProcessor(
			processor.Name,
			processor.SQL,
			processor.Runners,
			processor.ClusterName,
			processor.Namespace,
			processor.Pipeline,
			processor.AppID); err != nil {

			golog.Errorf("Error creating processor from file [%s/%s]. [%s]", loadpath, file.Name(), err.Error())
			return err
		}

		golog.Infof("Created processor from [%s/%s]", loadpath, file.Name())
	}

	return nil
}
