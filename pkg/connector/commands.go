package connector

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/spf13/cobra"
)

// NewConnectorsCommand creates the `connectors` command
func NewConnectorsCommand() *cobra.Command {
	var (
		clusterName string

		namesOnly bool // if true then print only the connector names and not the details as json.
		unwrap    bool // if true and namesOnly is true then print just the connectors names as a list of strings.

		showSupportedOnly bool // if true then show only the supported Kafka Connectors (static info).
	)

	root := &cobra.Command{
		Use:              "connectors",
		Short:            "List of active connectors' names",
		Aliases:          []string{"connect"},
		Example:          `connectors [--supported] or connectors --cluster-name="cluster_name" or --cluster-name="*"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if showSupportedOnly {
				connectorsInfo, err := config.Client.GetSupportedConnectors()
				if err != nil {
					golog.Errorf("Failed to find connector pugins. [%s]", err.Error())
					return err
				}

				sort.Slice(connectorsInfo, func(i, j int) bool {
					return connectorsInfo[i].Name < connectorsInfo[j].Name
				})

				if namesOnly {
					var names []string
					for _, c := range connectorsInfo {
						names = append(names, c.Name)
					}

					if unwrap {
						for _, name := range names {
							fmt.Fprintln(cmd.OutOrStdout(), name)
						}
						return nil
					}

					return bite.PrintObject(cmd, bite.OutlineStringResults(cmd, "name", names))
				}

				return bite.PrintObject(cmd, connectorsInfo)
			}

			connectorNames := make(map[string][]string) // clusterName:[] connectors names.

			if clusterName == "*" || clusterName == "" {
				// if * then no clusterName given,
				// fetch the connectors from all known clusters and print them.
				clusters, err := config.Client.GetConnectClusters()
				if err != nil {
					return err
				}
				for _, clusterName := range clusters {
					clusterConnectorsNames, err := config.Client.GetConnectors(clusterName)
					if err != nil {
						return err
					}
					connectorNames[clusterName] = append(connectorNames[clusterName], clusterConnectorsNames...)
				}
			} else {
				names, err := config.Client.GetConnectors(clusterName)
				if err != nil {
					golog.Errorf("Failed to find connectors in cluster [%s]. [%s]", clusterName, err.Error())
					return err
				}

				connectorNames[clusterName] = names
			}

			if namesOnly {
				var names []string
				for _, cNames := range connectorNames {
					names = append(names, cNames...)
				}

				sort.Strings(names)

				if unwrap {
					for _, name := range names {
						fmt.Fprintln(cmd.OutOrStdout(), name)
					}
					return nil
				}

				// return printJSON(cmd, outlineStringResults("name", names))
				return bite.PrintObject(cmd, bite.OutlineStringResults(cmd, "name", names))
			}

			// if json output requested, create a json object which is the group of cluster:[]connectors and print as json.

			// if table mode view, select all connectors as a list,
			// do not make the group "visual" based on cluster name here (still they are grouped).
			var connectors []api.Connector
			for cluster, names := range connectorNames {
				sort.Strings(names)
				for _, name := range names {
					connector, err := config.Client.GetConnector(cluster, name)
					if err != nil {
						fmt.Fprintf(cmd.OutOrStderr(), "get connector error: [%v]\n", err)
						continue
					}

					connectors = append(connectors, connector)
				}
			}

			return bite.PrintObject(cmd, connectors)
		},
	}

	root.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)
	root.Flags().BoolVar(&namesOnly, "names", false, `Print connector names only`)
	root.Flags().BoolVar(&unwrap, "unwrap", false, "--unwrap")
	root.Flags().BoolVar(&showSupportedOnly, "supported", false, "List all the supported Kafka Connectors instead of the currently deployed")

	bite.CanPrintJSON(root)

	// plugins subcommand.
	root.AddCommand(NewGetConnectorsPluginsCommand())

	// clusters subcommand.
	root.AddCommand(NewGetConnectorsClustersCommand())

	return root
}

// NewGetConnectorsPluginsCommand creates the `connectors plugins` command
func NewGetConnectorsPluginsCommand() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:           "plugins",
		Short:         "List of available connectors' plugins",
		Example:       `connectors plugins --cluster-name="cluster_name"`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var plugins []api.ConnectorPlugin

			if clusterName == "*" {
				// if * then no clusterName given, fetch the plugins from all known clusters and print them.
				clusters, err := config.Client.GetConnectClusters()
				if err != nil {
					golog.Errorf("Failed to connect clusters. [%s]", err.Error())
					return err
				}

				for _, clusterName := range clusters {
					clusterPlugins, err := config.Client.GetConnectorPlugins(clusterName)
					if err != nil {
						golog.Errorf("Failed to find connector pugins. [%s]", err.Error())
						return err
					}
					plugins = append(plugins, clusterPlugins...)
				}
			} else {
				var err error
				plugins, err = config.Client.GetConnectorPlugins(clusterName)
				if err != nil {
					return err
				}
			}

			for i, p := range plugins {
				if p.Version == "null" || p.Version == "" {
					plugins[i].Version = "X.X.X"
				}
			}

			return bite.PrintObject(cmd, plugins)
		},
	}

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)

	bite.CanPrintJSON(cmd)

	return cmd
}

// NewGetConnectorsClustersCommand creates the `connectors plugins` command
func NewGetConnectorsClustersCommand() *cobra.Command {
	var (
		namesOnly bool
		noNewLine bool // matters when namesOnly is true.
	)

	cmd := &cobra.Command{
		Use:           "clusters",
		Short:         "List of available connectors' clusters",
		Example:       `connectors clusters`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			clusters, err := config.Client.GetConnectClusters()
			if err != nil {
				golog.Errorf("Failed to read connect clusters. [%s]", err.Error())
				return err
			}

			sort.Slice(clusters, func(i, j int) bool {
				return clusters[i] < clusters[j]
			})

			if namesOnly {
				var b strings.Builder

				for i, clusterName := range clusters {
					b.WriteString(fmt.Sprintf("%s", clusterName))
					if !noNewLine && len(clusters)-1 != i {
						// add new line if enabled and not last, note that we use the fmt.Println below
						// even if newLine is disabled (for unix terminals mostly).
						b.WriteString("\n")
					}
				}

				_, err = fmt.Fprintln(cmd.OutOrStdout(), b.String())
				return err
			}

			// return printJSON(cmd, clusters)
			return bite.PrintObject(cmd, clusters)
		},
	}

	cmd.Flags().BoolVar(&namesOnly, "names", false, `Print connector names only`)
	cmd.Flags().BoolVar(&noNewLine, "no-newline", false, "Remove line breakers between string output, if --names is passed")
	bite.CanPrintJSON(cmd)

	return cmd
}

// NewConnectorGroupCommand creates the `connector` command
func NewConnectorGroupCommand() *cobra.Command {
	var clusterName, name string
	root := &cobra.Command{
		Use:              "connector",
		Short:            "Get information about a particular connector based on its name",
		Example:          `connector --cluster-name="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			connector, err := config.Client.GetConnector(clusterName, name)
			if err != nil {
				golog.Errorf("Failed to find connector [%s] in cluster [%s]. [%s]", name, clusterName, err.Error())
				return err
			}

			// return printJSON(cmd, connector)
			return bite.PrintObject(cmd, connector)
		},
	}

	bite.CanPrintJSON(root)

	root.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)
	root.Flags().StringVar(&name, "name", "", `Connector name`)

	// subcommands.
	root.AddCommand(NewConnectorCreateCommand())
	root.AddCommand(NewConnectorUpdateCommand())
	root.AddCommand(NewConnectorGetConfigCommand())
	root.AddCommand(NewConnectorGetStatusCommand())
	root.AddCommand(NewConnectorPauseCommand())
	root.AddCommand(NewConnectorResumeCommand())
	root.AddCommand(NewConnectorRestartCommand())
	root.AddCommand(NewConnectorGetTasksCommand())
	root.AddCommand(NewConnectorDeleteCommand())
	// connector.task subcommands.
	root.AddCommand(NewConnectorTaskGroupCommand())

	return root
}

// NewConnectorCreateCommand creates the `connector create` command
func NewConnectorCreateCommand() *cobra.Command {
	var (
		configRaw string
		connector = api.CreateUpdateConnectorPayload{Config: make(api.ConnectorConfig)}
	)

	cmd := &cobra.Command{
		Use:              "create",
		Short:            "Create a new connector",
		Example:          `connector create --cluster-name="cluster_name" --name="connector_name" --configs="{\"key\": \"value\"}" or connector create ./connector.yml`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := connector.ApplyAndValidateName(); err != nil {
				return err
			}

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": connector.ClusterName, "name": connector.Name}); err != nil {
				return err
			}

			if configRaw != "" {
				if err := bite.TryReadFile(configRaw, &connector.Config); err != nil {
					// from flag as json.
					if err = json.Unmarshal([]byte(configRaw), &connector.Config); err != nil {
						return fmt.Errorf("Unable to unmarshal the config: [%v]", err)
					}
				}
			}

			_, err := config.Client.CreateConnector(connector.ClusterName, connector.Name, connector.Config)
			if err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Connector [%s] created", connector.Name)
		},
	}

	cmd.Flags().StringVar(&connector.ClusterName, "cluster-name", "", `Connect cluster name`)
	cmd.Flags().StringVar(&connector.Name, "name", "", `Connector name`)
	cmd.Flags().StringVar(&configRaw, "configs", "", `Connector config .e.g."{\"key\": \"value\"}"`) // --config conflicts with the global flag.
	bite.CanBeSilent(cmd)

	bite.ShouldTryLoadFile(cmd, &connector)

	return cmd
}

// NewConnectorUpdateCommand creates the `connector update` command
func NewConnectorUpdateCommand() *cobra.Command {
	var (
		configRaw string
		connector = api.CreateUpdateConnectorPayload{Config: make(api.ConnectorConfig)}
	)

	cmd := &cobra.Command{
		Use:              "update",
		Short:            "Update a connector's configuration",
		Example:          `connector update --cluster-name="cluster_name" --name="connector_name" --configs="{\"key\": \"value\"}" or connector update ./connector.yml`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := connector.ApplyAndValidateName(); err != nil {
				return err
			}

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": connector.ClusterName, "name": connector.Name}); err != nil {
				return err
			}

			if configRaw != "" {
				if err := bite.TryReadFile(configRaw, &connector.Config); err != nil {
					// from flag as json.
					if err = json.Unmarshal([]byte(configRaw), &connector.Config); err != nil {
						return fmt.Errorf("Unable to unmarshal the config: [%v]", err)
					}
				}
			}

			// for any case.
			existingConnector, err := config.Client.GetConnector(connector.ClusterName, connector.Name)
			if err != nil {
				bite.FriendlyError(cmd, pkg.ErrResourceNotFoundMessage, "connector [%s:%s] does not exist", connector.ClusterName, connector.Name)
				return err
			}

			if existingConnector.Config != nil {
				if existingNameValue := existingConnector.Config["name"]; existingNameValue != connector.Name {
					return fmt.Errorf(`Connector config["name"] [%s] does not match with the existing one [%s]`, connector.Name, existingNameValue)
				}
			}

			updatedConnector, err := config.Client.UpdateConnector(connector.ClusterName, connector.Name, connector.Config)
			if err != nil {
				// bite.FriendlyError(cmd, errResourceNotAccessibleMessage, "unable to update connector [%s:%s], the action requires 'Write' permissions", connector.ClusterName, connector.Name)
				return err
			}

			//  why we print it back based on the --silent? Because of the connector.Tasks.
			if !bite.ExpectsFeedback(cmd) {
				bite.PrintInfo(cmd, "Connector [%s] updated\n\n", connector.Name)
				return bite.PrintObject(cmd, updatedConnector)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&connector.ClusterName, "cluster-name", "", `Connect cluster name`)
	cmd.Flags().StringVar(&connector.Name, "name", "", `Connector name`)
	cmd.Flags().StringVar(&configRaw, "configs", "", `Connector configs .e.g. "{\"key\": \"value\"}"`)

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)

	bite.ShouldTryLoadFile(cmd, &connector)

	return cmd
}

// NewConnectorGetConfigCommand creates the `connector config` command
func NewConnectorGetConfigCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "config",
		Short:            "Get connector config",
		Example:          `connector config --cluster-name="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			cfg, err := config.Client.GetConnectorConfig(clusterName, name)
			if err != nil {
				golog.Errorf("Failed to retrieve connector config for [%s] in cluster [%s]. [%s]", name, clusterName, err.Error())
				return err
			}

			// return printJSON(cmd, cfg)
			return bite.PrintObject(cmd, cfg)
		},
	}

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)
	cmd.Flags().StringVar(&name, "name", "", `Connector name`)

	bite.CanPrintJSON(cmd)

	return cmd
}

// NewConnectorGetStatusCommand creates the `connector status` command
func NewConnectorGetStatusCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "status",
		Short:            "Get connector status",
		Example:          `connector status --cluster-name="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			cs, err := config.Client.GetConnectorStatus(clusterName, name)
			if err != nil {
				golog.Errorf("Failed to retrieve connector status for [%s] in cluster [%s]. [%s]", name, clusterName, err.Error())
				return err
			}

			// return printJSON(cmd, cs)
			return bite.PrintObject(cmd, cs)
		},
	}

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name"`)
	cmd.Flags().StringVar(&name, "name", "", `Connector name`)

	bite.CanPrintJSON(cmd)

	return cmd
}

// NewConnectorPauseCommand creates the `connector pause` command
func NewConnectorPauseCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "pause",
		Short:            "Pause a connector",
		Example:          `connector pause --cluster-name="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			if err := config.Client.PauseConnector(clusterName, name); err != nil {
				golog.Errorf("Failed to pause connector [%s] in cluster [%s]. [%s]", name, clusterName, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Connector [%s:%s] paused", clusterName, name)
		},
	}

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)
	cmd.Flags().StringVar(&name, "name", "", `Connector name`)
	bite.CanBeSilent(cmd)

	return cmd
}

// NewConnectorResumeCommand creates the `connector resume` command
func NewConnectorResumeCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "resume",
		Short:            "Resume a paused connector",
		Example:          `connector resume --cluster-name="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			if err := config.Client.ResumeConnector(clusterName, name); err != nil {
				golog.Errorf("Failed to resume connector [%s] in cluster [%s]. [%s]", name, clusterName, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Connector [%s:%s] resumed", clusterName, name)
		},
	}

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)
	cmd.Flags().StringVar(&name, "name", "", `Connector name`)
	bite.CanBeSilent(cmd)

	return cmd
}

// NewConnectorRestartCommand creates the `connector restart` command
func NewConnectorRestartCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "restart",
		Short:            "Restart a connector",
		Example:          `connector restart --cluster-name="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			if err := config.Client.RestartConnector(clusterName, name); err != nil {
				golog.Errorf("Failed to restart connector [%s] in cluster [%s]. [%s]", name, clusterName, err.Error())
			}

			return bite.PrintInfo(cmd, "Connector [%s:%s] restarted", clusterName, name)
		},
	}

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name"`)
	cmd.Flags().StringVar(&name, "name", "", `Connector name`)
	bite.CanBeSilent(cmd)

	return cmd
}

// NewConnectorGetTasksCommand creates the `connector tasks` command
func NewConnectorGetTasksCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "tasks",
		Short:            "List of connector tasks",
		Example:          `connector tasks --cluster-name="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			tasksMap, err := config.Client.GetConnectorTasks(clusterName, name)
			if err != nil {
				golog.Errorf("Failed to retrieve task status for connector [%s] in cluster [%s]. [%s]", name, clusterName, err.Error())
				return err
			}

			return bite.PrintObject(cmd, tasksMap)
		},
	}

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)
	cmd.Flags().StringVar(&name, "name", "", `Connector name`)

	bite.CanPrintJSON(cmd)

	return cmd
}

// NewConnectorTaskGroupCommand creates the `connector task` command
func NewConnectorTaskGroupCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "task",
		Short:            "Manage particular connector task, see connector task --help for details",
		Example:          `connector task status --cluster-name="cluster_name" --name="connector_name" --task=1`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	rootSub.AddCommand(NewConnectorGetCurrentTaskStatusCommand())
	rootSub.AddCommand(NewConnectorTaskRestartCommand())

	return rootSub
}

// NewConnectorGetCurrentTaskStatusCommand creates the `connector task status` command
func NewConnectorGetCurrentTaskStatusCommand() *cobra.Command {
	var (
		clusterName, name string
		taskID            int
	)
	cmd := &cobra.Command{
		Use:              "status",
		Short:            "Get current status of a task",
		Example:          `connector task status --cluster-name="cluster_name" --name="connector_name" --task=1`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			cst, err := config.Client.GetConnectorTaskStatus(clusterName, name, taskID)
			if err != nil {
				golog.Errorf("Failed to retrieve task [%d] for connector [%s] in cluster [%s]. [%s]", taskID, name, clusterName, err.Error())
				return err
			}

			// return printJSON(cmd, cst)
			return bite.PrintObject(cmd, cst)
		},
	}

	cmd.Flags().IntVar(&taskID, "task", 0, "--task=1 The Task ID")
	cmd.MarkFlagRequired("task")

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)
	cmd.Flags().StringVar(&name, "name", "", `Connector name`)

	bite.CanPrintJSON(cmd)

	return cmd
}

// NewConnectorTaskRestartCommand creates the `connector task restart` command
func NewConnectorTaskRestartCommand() *cobra.Command {
	var (
		clusterName, name string
		taskID            int
	)

	cmd := &cobra.Command{
		Use:              "restart",
		Short:            "Restart a connector task",
		Example:          `connector task restart --cluster-name="cluster_name" --name="connector_name" --task=1`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			if err := config.Client.RestartConnectorTask(clusterName, name, taskID); err != nil {
				golog.Errorf("Failed to restart task [%d] connector [%s] in cluster [%s]. [%s]", taskID, name, clusterName, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Connector task [%s:%s:%d] restarted", clusterName, name, taskID)
		},
	}

	cmd.Flags().IntVar(&taskID, "task", 0, "--task=1 The Task ID")
	cmd.MarkFlagRequired("task")
	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)
	cmd.Flags().StringVar(&name, "name", "", `Connector name`)
	bite.CanBeSilent(cmd)

	return cmd
}

// NewConnectorDeleteCommand creates the `connector task delete` command
func NewConnectorDeleteCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a running connector",
		Example:          `connector delete --cluster-name="" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"cluster-name": clusterName, "name": name}); err != nil {
				return err
			}

			if err := config.Client.DeleteConnector(clusterName, name); err != nil {
				golog.Errorf("Failed to delete connector [%s] in cluster [%s]. [%s]", name, clusterName, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Connector [%s:%s] deleted", clusterName, name)
		},
	}

	cmd.Flags().StringVar(&clusterName, "cluster-name", "", `Connect cluster name`)
	cmd.Flags().StringVar(&name, "name", "", `Connector name`)
	bite.CanBeSilent(cmd)

	return cmd
}
