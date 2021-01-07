package processor

import (
	"net/url"
	"sort"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewGetProcessorsCommand creates `processors` command
func NewGetProcessorsCommand() *cobra.Command {
	var name, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "processors",
		Short:            "List of all available processors",
		Example:          `processors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := config.Client.GetProcessors()
			if err != nil {
				golog.Errorf("Failed to retrieve processors. [%s]", err.Error())
				return err
			}

			mode, err := config.Client.GetExecutionMode()
			if err != nil {
				return err
			}

			sort.Slice(result.Streams, func(i, j int) bool {
				return result.Streams[i].Name < result.Streams[j].Name
			})

			var final []api.ProcessorStream

			for _, processor := range result.Streams {
				if mode == api.ExecutionModeConnect || mode == api.ExecutionModeKubernetes {
					if name != "" && processor.Name != name {
						continue
					}
				}

				if mode == api.ExecutionModeKubernetes {
					if namespace != "" && processor.Namespace != namespace {
						continue
					}
				}

				if clusterName != "" && clusterName != processor.ClusterName {
					continue
				}
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

	cmd.AddCommand(NewProcessorsLogsCommand())
	cmd.AddCommand(NewListDeploymentTargetsCommand())

	return cmd
}

//NewProcessorsLogsCommand creates `processors logs` command
func NewProcessorsLogsCommand() *cobra.Command {
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
				utils.RichLog(level, log)
				return nil
			}

			if err := config.Client.GetProcessorsLogs(clusterName, namespace, podName, follow, lines, handler); err != nil {
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

//NewProcessorGroupCommand creates `processor` command
func NewProcessorGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "processor",
		Short:            "Manage a processor based on the processor id; stop, start, update runners, delete or create a new processor",
		Example:          `processor pause --id="existing_processor_id" or processor create --name="processor_name" --sql="" --runners=1 --cluster-name="" --namespace="" pipeline=""`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	// subcommands
	root.AddCommand(NewProcessorViewCommand())
	root.AddCommand(NewProcessorCreateCommand())
	root.AddCommand(NewProcessorPauseCommand())
	root.AddCommand(NewProcessorResumeCommand())
	root.AddCommand(NewProcessorUpdateRunnersCommand())
	root.AddCommand(NewProcessorDeleteCommand())

	return root
}

//NewProcessorViewCommand creates `processor view` command
func NewProcessorViewCommand() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:              "view",
		Short:            "View a processor",
		Example:          `processor view --id cluster.namespace.name`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			processor, err := config.Client.GetProcessor(id)

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

//NewProcessorCreateCommand creates `processor create` command
func NewProcessorCreateCommand() *cobra.Command {
	// the processorName and sql are the required.
	var processor api.CreateProcessorPayload

	cmd := &cobra.Command{
		Use:              `create`,
		Short:            "Create a processor",
		Example:          `processor create --name="processor_name" --sql="" --runners=1 --cluster-name="" --namespace="" pipeline="" --id=""`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": processor.Name, "sql": processor.SQL}); err != nil {
				return err
			}

			err := config.Client.CreateProcessor(processor.Name, processor.SQL, processor.Runners, processor.ClusterName, processor.Namespace, processor.Pipeline, processor.ProcessorID)

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
	cmd.Flags().StringVar(&processor.ProcessorID, "id", "", `The processor identifier, it is used as the underlying Kafka consumer group`)

	bite.Prepend(cmd, bite.FileBind(&processor))
	bite.CanBeSilent(cmd)

	return cmd
}

//NewProcessorPauseCommand creates `processor pause` command
func NewProcessorPauseCommand() *cobra.Command {
	var processorID, processorName, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "stop",
		Short:            "Stop a processor",
		Example:          `processor stop --id="processor_id" (or --name="processor_name") --cluster-name="clusterName" --namespace="namespace"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := config.Client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			if err := config.Client.StopProcessor(identifier); err != nil {
				golog.Errorf("Failed to stop processor [%s]. [%s]", identifier, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Processor [%s] stopped", identifier)
		},
	}

	cmd.Flags().StringVar(&processorID, "id", "", "Processor ID to stop")
	cmd.Flags().StringVar(&processorName, "name", "", "Processor name to stop")
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Cluster name the processor is in`)
	cmd.Flags().StringVar(&namespace, "namespace", "", `Namespace the processor is in`)
	bite.CanBeSilent(cmd)

	return cmd
}

//NewProcessorResumeCommand creates `processor resume` command
func NewProcessorResumeCommand() *cobra.Command {
	var processorID, processorName, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "start",
		Short:            "Start a processor",
		Example:          `processor start --id="processor_id" (or --name="processor_name") --cluster-name="clusterName" --namespace="namespace"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := config.Client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			if err := config.Client.ResumeProcessor(identifier); err != nil {
				golog.Errorf("Failed to start processor [%s]. [%s]", identifier, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Processor [%s] started", identifier)
		},
	}

	cmd.Flags().StringVar(&processorID, "id", "", "Processor ID to start")
	cmd.Flags().StringVar(&processorName, "name", "", "Processor name to start")
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Cluster name the processor is in`)
	cmd.Flags().StringVar(&namespace, "namespace", "", `Namespace the processor is in`)
	bite.CanBeSilent(cmd)

	return cmd
}

//NewProcessorUpdateRunnersCommand creates `processor update` command
func NewProcessorUpdateRunnersCommand() *cobra.Command {

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

			identifier, err := config.Client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			if err := config.Client.UpdateProcessorRunners(identifier, runners); err != nil {
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

//NewProcessorDeleteCommand creates `processor delete` command
func NewProcessorDeleteCommand() *cobra.Command {
	var processorID, processorName, clusterName, namespace string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a processor",
		Example:          `processor delete --id="processor_id" (or --name="processor_name") --cluster-name="clusterName" --namespace="namespace"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			identifier, err := config.Client.LookupProcessorIdentifier(processorID, processorName, clusterName, namespace)
			if err != nil {
				return err
			}

			// delete the processor based on the identifier, based on the current running mode.
			if err := config.Client.DeleteProcessor(identifier); err != nil {
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

type (
	// ListTargetsResult output for listing
	ListTargetsResult struct {
		Type        string `json:"type" header:"Type"`
		ClusterName string `json:"clusterName" header:"Cluster"`
		Namespace   string `json:"namespace" header:"Namespace"`
		Version     string `json:"version" header:"Version"`
	}
)

//NewListDeploymentTargetsCommand lists the available deployment targets
func NewListDeploymentTargetsCommand() *cobra.Command {
	var clusterName, targetType string

	cmd := &cobra.Command{
		Use:   "targets",
		Short: "List available target clusters to deploy to Kubernetes",
		Example: `
processors targets --target-type kubernetes --cluster-name="clusterName"
processors targets --target-type connect`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			targets, err := config.Client.GetDeploymentTargets()
			if err != nil {
				return err
			}

			var results []ListTargetsResult
			targetType = strings.ToLower(targetType)

			if targetType == "" || targetType == "kubernetes" {
				for _, kt := range targets.Kubernetes {
					if clusterName != "" && clusterName != kt.Cluster {
						continue
					}

					for _, ns := range kt.Namespaces {
						results = append(results, ListTargetsResult{"Kubernetes", kt.Cluster, ns, kt.Version})
					}
				}
			}

			if targetType == "" || targetType == "connect" {
				for _, connect := range targets.Connect {
					if clusterName != "" && clusterName != connect.Cluster {
						continue
					}

					results = append(results, ListTargetsResult{"Connect", connect.Cluster, "", connect.Version})
				}
			}

			return bite.PrintObject(cmd, results)
		},
	}

	cmd.Flags().StringVar(&targetType, "target-type", "", `Target type to filter by, e.g. Kubernetes or Connect.`)
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Cluster name to filter by`)
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)

	return cmd
}
