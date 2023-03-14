package imports

import (
	"fmt"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"

	"github.com/kataras/golog"
	"github.com/spf13/cobra"
)

// NewImportProcessorsCommand import processors command
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
	files, err := utils.FindFiles(loadpath)
	if err != nil {
		return err
	}

	processors, err := client.GetProcessors()

	if err != nil {
		golog.Errorf("Failed to retrieve processors. [%s]", err.Error())
	}

IterateImportFiles:
	for _, file := range files {

		var processor api.CreateProcessorFilePayload

		if err := load(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &processor); err != nil {
			return err
		}

		for _, p := range processors.Streams {
			if processor.Name != p.Name ||
				processor.ClusterName != p.ClusterName ||
				processor.Namespace != p.Namespace {
				continue
			}

			if processor.Runners == p.Runners {
				golog.Warnf("Processor [%s] from file [%s/%s] already exists", p.ID, loadpath, file.Name())
				// Iterate next file from 'files'
				continue IterateImportFiles
			}
			//scale
			if err := client.UpdateProcessorRunners(p.ID, processor.Runners); err != nil {
				golog.Errorf("Error scaling processor [%s] from file [%s/%s]. [%s]", p.ID, loadpath, file.Name(), err.Error())
				return err
			}
			golog.Infof("Scaled processor [%s] from file [%s/%s] from [%d] to [%d]", p.ID, loadpath, file.Name(), p.Runners, processor.Runners)
			return nil

		}

		if err := client.CreateProcessor(
			processor.Name,
			processor.SQL,
			processor.Runners,
			processor.ClusterName,
			processor.Namespace,
			processor.Pipeline,
			processor.ProcessorID); err != nil {

			golog.Errorf("Error creating processor from file [%s/%s]. [%s]", loadpath, file.Name(), err.Error())
			return err
		}

		golog.Infof("Created processor from [%s/%s]", loadpath, file.Name())
	}

	return nil
}
