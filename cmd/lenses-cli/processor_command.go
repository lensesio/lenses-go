package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newGetProcessorsCommand())
	rootCmd.AddCommand(newProcessorGroupCommand())
}

func newGetProcessorsCommand() *cobra.Command {
	var name, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "processors",
		Short:            "List of all available processors",
		Example:          exampleString(`processors`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := client.GetProcessors()
			if err != nil {
				return err
			}

			mode, err := client.GetExecutionMode()
			if err != nil {
				return err
			}

			sort.Slice(result.Streams, func(i, j int) bool {
				return result.Streams[i].Name < result.Streams[j].Name
			})

			for _, processor := range result.Streams {
				if mode == lenses.ExecutionModeConnect || mode == lenses.ExecutionModeKubernetes {

					if name != "" {
						if processor.Name != name {
							continue
						}
					}

					if clusterName != "" {
						if processor.ClusterName != clusterName {
							continue
						}
					}
				}

				if mode == lenses.ExecutionModeKubernetes {
					if namespace != "" {
						if processor.Namespace != namespace {
							continue
						}
					}
				}

				processor.SQL = strings.Replace(processor.SQL, "\n", "", -1)
				processor.SQL = strings.Replace(processor.SQL, "   ", "", -1)
			}

			return printJSON(cmd, result.Streams)
		},
	}

	// select by name (maybe more than one in CONNECT and KUBERNETES mode) and cluster and namespace or name or cluster or namespace only.
	cmd.Flags().StringVar(&name, "name", "", "--name=processorName select by processor name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&clusterName, "clusterName", "", "--clusterName=clusterName select by cluster name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&namespace, "namespace", "", "--namespace=namespace select by namespace, available only in KUBERNETES mode")

	// example: lenses-cli processors --query="[?ClusterName == 'IN_PROC'].Name | sort(@) | {Processor_Names_IN_PROC: join(', ', @)}"
	canPrintJSON(cmd)

	return cmd
}

func newProcessorGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "processor",
		Short:            "Work with a particular processor based on the processor id; pause, resume, update runners, delete or create a new processor",
		Example:          exampleString(`processor pause --id="existing_processor_id" or processor create --name="processor_name" --sql="" --runners=1 --clusterName="" --namespace="" pipeline=""`),
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	// subcommands
	root.AddCommand(newProcessorCreateCommand())
	root.AddCommand(newProcessorPauseCommand())
	root.AddCommand(newProcessorResumeCommand())
	root.AddCommand(newProcessorUpdateRunnersCommand())
	root.AddCommand(newProcessorDeleteCommand())

	return root
}

func newProcessorCreateCommand() *cobra.Command {
	// the processorName and sql are the required.
	var processor lenses.CreateProcessorPayload

	cmd := &cobra.Command{
		Use:              "create",
		Short:            "Create a processor",
		Example:          exampleString(`processor create --name="processor_name" --sql="" --runners=1 --clusterName="" --namespace="" pipeline=""`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"name": processor.Name, "sql": processor.SQL}); err != nil {
				return err
			}

			err := client.CreateProcessor(processor.Name, processor.SQL, processor.Runners, processor.ClusterName, processor.Namespace, processor.Pipeline)

			if err != nil {
				return err
			}

			return echo(cmd, "Processor %s created", processor.Name)
		},
	}

	cmd.Flags().StringVar(&processor.Name, "name", "", "--name=processorName")
	cmd.Flags().StringVar(&processor.ClusterName, "clusterName", "", `--clusterName="clusterName"`)
	cmd.Flags().StringVar(&processor.Namespace, "namespace", "", `--namespace="namespace"`)
	cmd.Flags().StringVar(&processor.SQL, "sql", "", `--sql="SET autocreate=true;INSERT INTO topic1 SELECT * FROM topicA WHERE  _ktype='BYTES' AND _vtype='AVRO'"`)
	cmd.Flags().IntVar(&processor.Runners, "runners", 1, "--runners=1")
	cmd.Flags().StringVar(&processor.Pipeline, "pipeline", "", `--pipeline="pipeline" a label to apply to kubernetes processors, defaults to processor name`)

	shouldTryLoadFile(cmd, &processor)
	canBeSilent(cmd)

	return cmd
}

func newProcessorPauseCommand() *cobra.Command {
	var processorID, processorName, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "pause",
		Short:            "Pause a processor",
		Example:          exampleString(`processor pause --id="processor_id" (or --name="processor_name") --clusterName="clusterName" --namespace="namespace"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			if err := client.PauseProcessor(identifier); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to pause, processor '%s' does not exist", identifier)
				return err
			}

			return echo(cmd, "Processor %s paused", identifier)
		},
	}

	cmd.Flags().String("id", "", "--id=processor_1")
	cmd.Flags().String("name", "", "--name=processorName")
	cmd.Flags().String("clusterName", "", `--clusterName="clusterName"`)
	cmd.Flags().String("namespace", "", `--namespace="namespace"`)
	canBeSilent(cmd)

	return cmd
}

func newProcessorResumeCommand() *cobra.Command {
	var processorID, processorName, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "resume",
		Short:            "Resume a processor",
		Example:          exampleString(`processor resume --id="processor_id" (or --name="processor_name") --clusterName="clusterName" --namespace="namespace"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			if err := client.ResumeProcessor(identifier); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to resume, processor '%s' does not exist", identifier)
				return err
			}

			return echo(cmd, "Processor %s resumed", identifier)
		},
	}

	cmd.Flags().String("id", "", "--id=processor_1")
	cmd.Flags().String("name", "", "--name=processorName")
	cmd.Flags().String("clusterName", "", `--clusterName="clusterName"`)
	cmd.Flags().String("namespace", "", `--namespace="namespace"`)
	canBeSilent(cmd)

	return cmd
}

func newProcessorUpdateRunnersCommand() *cobra.Command {

	var (
		runners                                            int
		processorID, processorName, clusterName, namespace string
	)

	cmd := &cobra.Command{
		Use:              "update",
		Short:            "Update processor runners",
		Example:          exampleString(`processor update --id="processor_id" (or --name="processor_name") --clusterName="clusterName" --namespace="namespace"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"runners": runners}); err != nil {
				return err
			}

			identifier, err := client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			if err := client.UpdateProcessorRunners(identifier, runners); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to scale to %d runners, processor '%s' does not exist", runners, identifier)
				return err
			}

			return echo(cmd, "Processor %s scaled", identifier)
		},
	}

	cmd.Flags().IntVar(&runners, "runners", 1, "--runners=2")

	cmd.Flags().String("id", "", "--id=processor_1")
	cmd.Flags().String("name", "", "--name=processorName")
	cmd.Flags().String("clusterName", "", `--clusterName="clusterName"`)
	cmd.Flags().String("namespace", "", `--namespace="namespace"`)
	canBeSilent(cmd)

	return cmd
}

func newProcessorDeleteCommand() *cobra.Command {
	var processorID, processorName, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a processor",
		Example:          exampleString(`processor delete --id="processor_id" (or --name="processor_name") --clusterName="clusterName" --namespace="namespace"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			// delete the processor based on the identifier, based on the current running mode.
			if err := client.DeleteProcessor(identifier); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to delete, processor '%s' does not exist", identifier)
				return err
			}

			return echo(cmd, "Processor %s deleted", identifier)
		},
	}

	// On CONNECT and IN_PROC and KUBERNETES modes can accept name or id (parent command flags).
	// On KUBERNETES mode clusterName and namespace should be passed (parent command flags) .

	cmd.Flags().String("id", "", "--id=processor_1")
	cmd.Flags().String("name", "", "--name=processorName")
	cmd.Flags().String("clusterName", "", `--clusterName="clusterName"`)
	cmd.Flags().String("namespace", "", `--namespace="namespace"`)
	canBeSilent(cmd)

	return cmd
}
