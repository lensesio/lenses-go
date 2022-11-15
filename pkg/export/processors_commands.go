package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportProcessorsCommand creates `export processors` command
func NewExportProcessorsCommand() *cobra.Command {
	var name, cluster, namespace, id string

	cmd := &cobra.Command{
		Use:              "processors",
		Short:            "export processors",
		Example:          `export processors --resource-name my-processor`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			setExecutionMode(client)
			checkFileFlags(cmd)
			if err := writeProcessors(cmd, client, id, cluster, namespace, name); err != nil {
				golog.Errorf("Error writing processors. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract dependencies, topics, acls, quotas, alerts")
	cmd.Flags().StringVar(&name, "resource-name", "", "The processor name to export")
	cmd.Flags().StringVar(&cluster, "cluster-name", "", "Select by cluster name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Select by namespace, available only in KUBERNETES mode")
	cmd.Flags().StringVar(&id, "id", "", "ID of the processor to export")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Processor with the prefix in the name only")

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeProcessors(cmd *cobra.Command, client *api.Client, id, cluster, namespace, name string) error {

	if mode == api.ExecutionModeInProcess {
		cluster = "IN-PROC"
		namespace = "Lenses"
	}
	processors, err := client.GetProcessors()
	if err != nil {
		return err
	}

	for _, processor := range processors.Streams {
		if id != "" && id != processor.ID {
			continue
		} else {
			if cluster != "" && cluster != processor.ClusterName {
				continue
			}

			if namespace != "" && namespace != processor.Namespace {
				continue
			}

			if name != "" && name != processor.Name {
				continue
			}

			if prefix != "" && !strings.HasPrefix(processor.Name, prefix) {
				continue
			}
		}
		request := processor.ProcessorAsFile()

		output := strings.ToUpper(bite.GetOutPutFlag(cmd))

		if output == "TABLE" {
			output = "YAML"
		}

		var fileName string

		if mode == api.ExecutionModeInProcess {
			fileName = fmt.Sprintf("processor-%s.%s", strings.ToLower(processor.Name), strings.ToLower(output))
		} else if mode == api.ExecutionModeConnect {
			fileName = fmt.Sprintf("processor-%s-%s.%s", strings.ToLower(processor.ClusterName), strings.ToLower(processor.Name), strings.ToLower(output))
		} else {
			fileName = fmt.Sprintf("processor-%s-%s-%s.%s", strings.ToLower(processor.ClusterName), strings.ToLower(processor.Namespace), strings.ToLower(processor.Name), strings.ToLower(output))
		}

		// trim so the yaml is a multiline string
		request.SQL = strings.TrimSpace(request.SQL)
		request.SQL = strings.Replace(request.SQL, "\t", "  ", -1)
		request.SQL = strings.Replace(request.SQL, " \n", "\n", -1)

		if err := utils.WriteFile(landscapeDir, pkg.SQLPath, fileName, output, request); err != nil {
			return err
		}
		if dependents {
			handleDependents(cmd, client, processor.ID)
		}

		exportPath := fmt.Sprintf("%s/%s/%s", landscapeDir, pkg.SQLPath, fileName)
		fmt.Printf("processor '%s' has been successfully exported at %s\n", request.Name, exportPath)
	}

	return nil
}
