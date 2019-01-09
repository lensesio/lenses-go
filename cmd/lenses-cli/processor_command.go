package main

import (
	"net/url"
	"sort"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newGetProcessorsCommand())
	app.AddCommand(newProcessorGroupCommand())
}

func newGetProcessorsCommand() *cobra.Command {
	var name, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "processors",
		Short:            "List of all available processors",
		Example:          `processors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			result, err := client.GetProcessors()
			if err != nil {
				golog.Errorf("Failed to retrieve processors. [%s]", err.Error())
				return err
			}

			mode, err := client.GetExecutionMode()
			if err != nil {
				return err
			}

			if mode == lenses.ExecutionModeInProcess {
				clusterName = "IN_PROC"
				namespace = "lenses"
			}

			sort.Slice(result.Streams, func(i, j int) bool {
				return result.Streams[i].Name < result.Streams[j].Name
			})

			var final []lenses.ProcessorStream

			for _, processor := range result.Streams {

				if clusterName != "" && clusterName != processor.ClusterName {
					continue
				}

				if namespace != "" && namespace != processor.Namespace {
					continue
				}

				if name != "" && name != processor.Name {
					continue
				}

				//processor.SQL = strings.Replace(processor.SQL, "\n", "", -1)
				//processor.SQL = strings.Replace(processor.SQL, "   ", "", -1)

				final = append(final, processor)
			}

			return bite.PrintObject(cmd, final)
		},
	}

	// select by name (maybe more than one in CONNECT and KUBERNETES mode) and cluster and namespace or name or cluster or namespace only.
	cmd.Flags().StringVar(&name, "name", "", "Select by processor name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", "Select by cluster name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Select by namespace, available only in KUBERNETES mode")
	// example: lenses-cli processors --query="[?ClusterName == 'IN_PROC'].Name | sort(@) | {Processor_Names_IN_PROC: join(', ', @)}"
	bite.CanPrintJSON(cmd)

	cmd.AddCommand(newProcessorsLogsCommand())

	return cmd
}

func newProcessorsLogsCommand() *cobra.Command {
	var (
		clusterName, podName, namespace string
		follow                          bool
		lines                           int
	)

	cmd := &cobra.Command{
		Use:              "logs",
		Short:            "Retrieve LSQL Processor logs. Available only in KUBERNETES execution mode",
		Example:          `processors logs --cluster-name=cluster-name --namespace=nameSpace --podName=runnerStateID [--follow --lines=50]`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "namespace": namespace, "podName": podName}); err != nil {
				return err
			}

			golog.SetTimeFormat("")
			handler := func(level, log string) error {
				log, _ = url.QueryUnescape(log) // for LSQL lines.
				richLog(level, log)
				return nil
			}

			if err := client.GetProcessorsLogs(clusterName, namespace, podName, follow, lines, handler); err != nil {
				golog.Errorf("Failed to retrieve logs for pod [%s]. [%s]", podName, err.Error())
				return err
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", "Select by cluster name, available only in KUBERNETES mode")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Select by namespace, available only in KUBERNETES mode")
	cmd.Flags().StringVar(&podName, "podName", "", "Kubernetes pod name to view the logs for")
	cmd.Flags().BoolVar(&follow, "follow", false, "Tail the log")
	cmd.Flags().IntVar(&lines, "lines", 100, "View the last n")
	return cmd
}

func newProcessorGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "processor",
		Short:            "Manage a processor based on the processor id; pause, resume, update runners, delete or create a new processor",
		Example:          `processor pause --id="existing_processor_id" or processor create --name="processor_name" --sql="" --runners=1 --cluster-name="" --namespace="" pipeline=""`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	// subcommands
	root.AddCommand(newProcessorViewCommand())
	root.AddCommand(newProcessorCreateCommand())
	root.AddCommand(newProcessorPauseCommand())
	root.AddCommand(newProcessorResumeCommand())
	root.AddCommand(newProcessorUpdateRunnersCommand())
	root.AddCommand(newProcessorDeleteCommand())

	return root
}

func newProcessorViewCommand() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:              "view",
		Short:            "View a processor",
		Example:          `processor view --id cluster.namespace.name`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			processor, err := client.GetProcessor(id)

			if err != nil {
				golog.Errorf("Failed to retrieve processor [%s]. [%s]", id, err.Error())
				return err
			}

			return bite.PrintObject(cmd, processor)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", `Processor id`)
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)

	return cmd
}

func newProcessorCreateCommand() *cobra.Command {
	// the processorName and sql are the required.
	var processor lenses.CreateProcessorPayload

	cmd := &cobra.Command{
		Use:              "create",
		Short:            "Create a processor",
		Example:          `processor create --name="processor_name" --sql="" --runners=1 --cluster-name="" --namespace="" pipeline=""`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": processor.Name, "sql": processor.SQL}); err != nil {
				return err
			}

			err := client.CreateProcessor(processor.Name, processor.SQL, processor.Runners, processor.ClusterName, processor.Namespace, processor.Pipeline)

			if err != nil {
				golog.Errorf("Failed to create processor [%s]. [%s]", processor.Name, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Processor [%s] created", processor.Name)
		},
	}

	cmd.Flags().StringVar(&processor.Name, "name", "", "Processor name")
	cmd.Flags().StringVar(&processor.ClusterName, "cluster-name", "", `Cluster to create the processor in`)
	cmd.Flags().StringVar(&processor.Namespace, "namespace", "", `Namespace to create the processor in`)
	cmd.Flags().StringVar(&processor.SQL, "sql", "", `Lenses SQL to run .e.g. sql="SET autocreate=true;INSERT INTO topic1 SELECT * FROM topicA"`)
	cmd.Flags().IntVar(&processor.Runners, "runners", 1, "Number of runners/instance to deploy")
	cmd.Flags().StringVar(&processor.Pipeline, "pipeline", "", `A label to apply to kubernetes processors, defaults to processor name`)

	bite.Prepend(cmd, bite.FileBind(&processor))
	bite.CanBeSilent(cmd)

	return cmd
}

func newProcessorPauseCommand() *cobra.Command {
	var processorID, processorName, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "pause",
		Short:            "Pause a processor",
		Example:          `processor pause --id="processor_id" (or --name="processor_name") --cluster-name="clusterName" --namespace="namespace"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			if err := client.PauseProcessor(identifier); err != nil {
				golog.Errorf("Failed to pause processor [%s]. [%s]", identifier, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Processor [%s] paused", identifier)
		},
	}

	cmd.Flags().StringVar(&processorID, "id", "", "Processor ID to pause")
	cmd.Flags().StringVar(&processorName, "name", "", "Processor name to pause")
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Cluster name the processor is in`)
	cmd.Flags().StringVar(&namespace, "namespace", "", `Namespace the processor is in`)
	bite.CanBeSilent(cmd)

	return cmd
}

func newProcessorResumeCommand() *cobra.Command {
	var processorID, processorName, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "resume",
		Short:            "Resume a processor",
		Example:          `processor resume --id="processor_id" (or --name="processor_name") --cluster-name="clusterName" --namespace="namespace"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			if err := client.ResumeProcessor(identifier); err != nil {
				golog.Errorf("Failed to resume processor [%s]. [%s]", identifier, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Processor [%s] resumed", identifier)
		},
	}

	cmd.Flags().StringVar(&processorID, "id", "", "Processor ID to resume")
	cmd.Flags().StringVar(&processorName, "name", "", "Processor name to resume")
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Cluster name the processor is in`)
	cmd.Flags().StringVar(&namespace, "namespace", "", `Namespace the processor is in`)
	bite.CanBeSilent(cmd)

	return cmd
}

func newProcessorUpdateRunnersCommand() *cobra.Command {

	var (
		runners                                            int
		processorID, processorName, clusterName, namespace string
	)

	cmd := &cobra.Command{
		Use:              "update",
		Aliases:          []string{"scale"},
		Short:            "Update processor runners",
		Example:          `processor update --id="processor_id" (or --name="processor_name") --cluster-name="clusterName" --namespace="namespace"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"runners": runners}); err != nil {
				return err
			}

			identifier, err := client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			if err := client.UpdateProcessorRunners(identifier, runners); err != nil {
				golog.Errorf("Failed to scale processor [%s] to [%d]. [%s]", identifier, runners, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Processor [%s] scaled", identifier)
		},
	}

	cmd.Flags().IntVar(&runners, "runners", 1, "Number of replicas to scale to")
	cmd.Flags().StringVar(&processorID, "id", "", "Processor ID to scale")
	cmd.Flags().StringVar(&processorName, "name", "", "Processor name to scale")
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Cluster name the processor is in`)
	cmd.Flags().StringVar(&namespace, "namespace", "", `Namespace the processor is in`)
	bite.CanBeSilent(cmd)

	return cmd
}

func newProcessorDeleteCommand() *cobra.Command {
	var processorID, processorName, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a processor",
		Example:          `processor delete --id="processor_id" (or --name="processor_name") --cluster-name="clusterName" --namespace="namespace"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			// delete the processor based on the identifier, based on the current running mode.
			if err := client.DeleteProcessor(identifier); err != nil {
				golog.Errorf("Failed to delete processor [%s]. [%s]", identifier, err.Error())
				return err
			}

			// change the printed value to the processor name if available.
			if processorName != "" {
				identifier = processorName
			}

			return bite.PrintInfo(cmd, "Processor [%s] deleted", identifier)
		},
	}

	// On CONNECT and IN_PROC and KUBERNETES modes can accept name or id (parent command flags).
	// On KUBERNETES mode clusterName and namespace should be passed (parent command flags) .

	cmd.Flags().StringVar(&processorID, "id", "", "Processor ID to delete")
	cmd.Flags().StringVar(&processorName, "name", "", "Processor name to delete")
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Cluster name the processor is in`)
	cmd.Flags().StringVar(&namespace, "namespace", "", `Namespace the processor is in`)
	bite.CanBeSilent(cmd)

	return cmd
}
