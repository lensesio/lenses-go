package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newConnectorsCommand())
	app.AddCommand(newConnectorGroupCommand())
}

func newConnectorsCommand() *cobra.Command {
	var (
		clusterName string

		namesOnly bool // if true then print only the connector names and not the details as json.
		unwrap    bool // if true and namesOnly is true then print just the connectors names as a list of strings.
	)

	root := &cobra.Command{
		Use:              "connectors",
		Short:            "List of active connectors' names",
		Aliases:          []string{"connect"},
		Example:          `connectors or connectors --clusterName="cluster_name" or --clusterName="*"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			connectorNames := make(map[string][]string) // clusterName:[] connectors names.

			if clusterName == "*" || clusterName == "" {
				// if * then no clusterName given,
				// fetch the connectors from all known clusters and print them.
				clusters, err := client.GetConnectClusters()
				if err != nil {
					return err
				}

				for _, cluster := range clusters {
					clusterConnectorsNames, err := client.GetConnectors(cluster.Name)
					if err != nil {
						return err
					}
					connectorNames[cluster.Name] = append(connectorNames[cluster.Name], clusterConnectorsNames...)
				}
			} else {
				names, err := client.GetConnectors(clusterName)
				if err != nil {
					bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to retrieve connectors, cluster with name '%s' does not exist", clusterName)
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

			connectors := make(map[string][]lenses.Connector, len(connectorNames))

			// else print the entire info.
			for cluster, names := range connectorNames {
				for _, name := range names {
					connector, err := client.GetConnector(cluster, name)
					if err != nil {
						fmt.Fprintf(cmd.OutOrStderr(), "get connector error: %v\n", err)
						continue
					}

					connectors[cluster] = append(connectors[cluster], connector)
				}
			}

			return bite.PrintJSON(cmd, connectors) // keep json?
		},
	}

	root.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName`)
	root.Flags().BoolVar(&namesOnly, "names", false, `--names`)
	root.Flags().BoolVar(&unwrap, "unwrap", false, "--unwrap")

	bite.CanPrintJSON(root)

	// plugins subcommand.
	root.AddCommand(newGetConnectorsPluginsCommand())

	// clusters subcommand.
	root.AddCommand(newGetConnectorsClustersCommand())

	return root
}

func newGetConnectorsPluginsCommand() *cobra.Command {
	var clusterName string

	cmd := &cobra.Command{
		Use:           "plugins",
		Short:         "List of available connectors' plugins",
		Example:       `connectors plugins --clusterName="cluster_name"`,
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var plugins []lenses.ConnectorPlugin

			if clusterName == "*" {
				// if * then no clusterName given, fetch the plugins from all known clusters and print them.
				clusters, err := client.GetConnectClusters()
				if err != nil {
					return err
				}

				for _, cluster := range clusters {
					clusterPlugins, err := client.GetConnectorPlugins(cluster.Name)
					if err != nil {
						return err
					}
					plugins = append(plugins, clusterPlugins...)
				}
			} else {
				var err error
				plugins, err = client.GetConnectorPlugins(clusterName)
				if err != nil {
					return err
				}
			}

			for _, p := range plugins {
				if p.Version == "null" || p.Version == "" {
					p.Version = "X.X.X"
				}
			}

			// return printJSON(cmd, plugins)
			return bite.PrintObject(cmd, plugins)
		},
	}

	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName`)

	bite.CanPrintJSON(cmd)

	return cmd
}

func newGetConnectorsClustersCommand() *cobra.Command {
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
			clusters, err := client.GetConnectClusters()
			if err != nil {
				return err
			}

			sort.Slice(clusters, func(i, j int) bool {
				return clusters[i].Name < clusters[j].Name
			})

			if namesOnly {
				var b strings.Builder

				for i, cl := range clusters {
					b.WriteString(fmt.Sprintf("%s", cl.Name))
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

	cmd.Flags().BoolVar(&namesOnly, "names", false, `--names`)
	cmd.Flags().BoolVar(&noNewLine, "no-newline", false, "--remove line breakers between string output, if --names is passed")
	bite.CanPrintJSON(cmd)

	return cmd
}

func newConnectorGroupCommand() *cobra.Command {
	var clusterName, name string

	root := &cobra.Command{
		Use:              "connector",
		Short:            "Get information about a particular connector based on its name",
		Example:          `connector --clusterName="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			connector, err := client.GetConnector(clusterName, name)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "connector '%s:%s' does not exist", clusterName, name)
				return err
			}

			// return printJSON(cmd, connector)
			return bite.PrintObject(cmd, connector)
		},
	}

	bite.CanPrintJSON(root)

	root.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	root.Flags().StringVar(&name, "name", "", `--name="connector_name"`)

	// subcommands.
	root.AddCommand(newConnectorCreateCommand())
	root.AddCommand(newConnectorUpdateCommand())
	root.AddCommand(newConnectorGetConfigCommand())
	root.AddCommand(newConnectorGetStatusCommand())
	root.AddCommand(newConnectorPauseCommand())
	root.AddCommand(newConnectorResumeCommand())
	root.AddCommand(newConnectorRestartCommand())
	root.AddCommand(newConnectorGetTasksCommand())
	root.AddCommand(newConnectorDeleteCommand())
	// connector.task subcommands.
	root.AddCommand(newConnectorTaskGroupCommand())

	return root
}

func newConnectorCreateCommand() *cobra.Command {
	var (
		configRaw string
		connector = lenses.CreateUpdateConnectorPayload{Config: make(lenses.ConnectorConfig)}
	)

	cmd := &cobra.Command{
		Use:              "create",
		Short:            "Create a new connector",
		Example:          `connector create --clusterName="cluster_name" --name="connector_name" --config="{\"key\": \"value\"}" or connector create ./connector.yml`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := connector.ApplyAndValidateName(); err != nil {
				return err
			}

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": connector.ClusterName, "name": connector.Name}); err != nil {
				return err
			}

			_, err := client.CreateConnector(connector.ClusterName, connector.Name, connector.Config)
			if err != nil {
				// give the exactly "low-level" message here, because we can't know if it's from the configuration
				// or if cluster does not exist here (<- *).
				errMsg := err.Error()
				bite.FriendlyError(cmd, errResourceNotGoodMessage, errMsg)
				bite.FriendlyError(cmd, errResourceNotFoundMessage, errMsg)
				bite.FriendlyError(cmd, errResourceNotAccessibleMessage, errMsg)
				return err
			}

			return bite.PrintInfo(cmd, "Connector %s created", connector.Name)
		},
	}

	cmd.Flags().StringVar(&connector.ClusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&connector.Name, "name", "", `--name="connector_name"`)
	cmd.Flags().StringVar(&configRaw, "config", "", `--config="{\"key\": \"value\"}"`)
	bite.CanBeSilent(cmd)

	bite.Prepend(cmd, bite.FileBind(&connector, bite.ElseBind(func() error {
		return bite.TryReadFile(configRaw, &connector.Config)
	})))

	return cmd
}

func newConnectorUpdateCommand() *cobra.Command { // almost the same as `newConnectorCreateCommand` but keep them separate, in future this may change.
	var (
		configRaw string
		connector = lenses.CreateUpdateConnectorPayload{Config: make(lenses.ConnectorConfig)}
	)

	cmd := &cobra.Command{
		Use:              "update",
		Short:            "Update a connector's configuration",
		Example:          `connector update --clusterName="cluster_name" --name="connector_name" --config="{\"key\": \"value\"}" or connector update ./connector.yml`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := connector.ApplyAndValidateName(); err != nil {
				return err
			}

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": connector.ClusterName, "name": connector.Name}); err != nil {
				return err
			}

			// for any case.
			existingConnector, err := client.GetConnector(connector.ClusterName, connector.Name)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "connector '%s:%s' does not exist", connector.ClusterName, connector.Name)
				return err
			}

			if existingConnector.Config != nil {
				if existingNameValue := existingConnector.Config["name"]; existingNameValue != connector.Name {
					return fmt.Errorf(`connector config["name"] '%s' does not match with the existing one '%s'`, connector.Name, existingNameValue)
				}
			}

			updatedConnector, err := client.UpdateConnector(connector.ClusterName, connector.Name, connector.Config)
			if err != nil {
				return err
			}

			//  why we print it back based on the --silent? Because of the connector.Tasks.
			if !bite.ExpectsFeedback(cmd) {
				bite.PrintInfo(cmd, "Connector %s updated\n\n", connector.Name)
				return bite.PrintObject(cmd, updatedConnector)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&connector.ClusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&connector.Name, "name", "", `--name="connector_name"`)
	cmd.Flags().StringVar(&configRaw, "config", "", `--config="{\"key\": \"value\"}"`)

	bite.Prepend(cmd, bite.FileBind(&connector, bite.ElseBind(func() error { return bite.TryReadFile(configRaw, &connector.Config) })))

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)

	return cmd
}

func newConnectorGetConfigCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "config",
		Short:            "Get connector config",
		Example:          `connector config --clusterName="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			cfg, err := client.GetConnectorConfig(clusterName, name)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to retrieve config, connector '%s:%s' does not exist", clusterName, name)
				return err
			}

			// return printJSON(cmd, cfg)
			return bite.PrintObject(cmd, cfg)
		},
	}

	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&name, "name", "", `--name="connector_name"`)

	bite.CanPrintJSON(cmd)

	return cmd
}

func newConnectorGetStatusCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "status",
		Short:            "Get connector status",
		Example:          `connector status --clusterName="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			cs, err := client.GetConnectorStatus(clusterName, name)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to retrieve status, connector '%s:%s' does not exist", clusterName, name)
				return err
			}

			// return printJSON(cmd, cs)
			return bite.PrintObject(cmd, cs)
		},
	}

	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&name, "name", "", `--name="connector_name"`)

	bite.CanPrintJSON(cmd)

	return cmd
}

func newConnectorPauseCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "pause",
		Short:            "Pause a connector",
		Example:          `connector pause --clusterName="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			if err := client.PauseConnector(clusterName, name); err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to pause, connector '%s:%s' does not exist", clusterName, name)
				return err
			}

			return bite.PrintInfo(cmd, "Connector %s:%s paused", clusterName, name)
		},
	}

	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&name, "name", "", `--name="connector_name"`)
	bite.CanBeSilent(cmd)

	return cmd
}

func newConnectorResumeCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "resume",
		Short:            "Resume a paused connector",
		Example:          `connector resume --clusterName="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			if err := client.ResumeConnector(clusterName, name); err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to resume, connector '%s:%s' does not exist", clusterName, name)
				return err
			}

			return bite.PrintInfo(cmd, "Connector %s:%s resumed", clusterName, name)
		},
	}

	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&name, "name", "", `--name="connector_name"`)
	bite.CanBeSilent(cmd)

	return cmd
}

func newConnectorRestartCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "restart",
		Short:            "Restart a connector",
		Example:          `connector restart --clusterName="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			if err := client.RestartConnector(clusterName, name); err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to restart, connector '%s:%s' does not exist", clusterName, name)
				return err
			}

			return bite.PrintInfo(cmd, "Connector %s:%s restarted", clusterName, name)
		},
	}

	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&name, "name", "", `--name="connector_name"`)
	bite.CanBeSilent(cmd)

	return cmd
}

func newConnectorGetTasksCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "tasks",
		Short:            "List of connector tasks",
		Example:          `connector tasks --clusterName="cluster_name" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			tasksMap, err := client.GetConnectorTasks(clusterName, name)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to retrieve tasks, connector '%s:%s' does not exist", clusterName, name)
				return err
			}

			// return printJSON(cmd, tasksMap)
			return bite.PrintObject(cmd, tasksMap)
		},
	}

	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&name, "name", "", `--name="connector_name"`)

	bite.CanPrintJSON(cmd)

	return cmd
}

func newConnectorTaskGroupCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "task",
		Short:            "Work with a particular connector task, see connector task --help for details",
		Example:          `connector task status --clusterName="cluster_name" --name="connector_name" --task=1`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	rootSub.AddCommand(newConnectorGetCurrentTaskStatusCommand())
	rootSub.AddCommand(newConnectorTaskRestartCommand())

	return rootSub
}

func newConnectorGetCurrentTaskStatusCommand() *cobra.Command {
	var (
		clusterName, name string
		taskID            int
	)

	cmd := &cobra.Command{
		Use:              "status",
		Short:            "Get current status of a task",
		Example:          `connector task status --clusterName="cluster_name" --name="connector_name" --task=1`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			cst, err := client.GetConnectorTaskStatus(clusterName, name, taskID)
			if err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "task does not exist")
				return err
			}

			// return printJSON(cmd, cst)
			return bite.PrintObject(cmd, cst)
		},
	}

	cmd.Flags().IntVar(&taskID, "task", 0, "--task=1 The Task ID")
	cmd.MarkFlagRequired("task")

	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&name, "name", "", `--name="connector_name"`)

	bite.CanPrintJSON(cmd)

	return cmd
}

func newConnectorTaskRestartCommand() *cobra.Command {
	var (
		clusterName, name string
		taskID            int
	)

	cmd := &cobra.Command{
		Use:              "restart",
		Short:            "Restart a connector task",
		Example:          `connector task restart --clusterName="cluster_name" --name="connector_name" --task=1`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			if err := client.RestartConnectorTask(clusterName, name, taskID); err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "task does not exist")
				return err
			}

			return bite.PrintInfo(cmd, "Connector task %s:%s:%d restarted", clusterName, name, taskID)
		},
	}

	cmd.Flags().IntVar(&taskID, "task", 0, "--task=1 The Task ID")
	cmd.MarkFlagRequired("task")
	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&name, "name", "", `--name="connector_name"`)
	bite.CanBeSilent(cmd)

	return cmd
}

func newConnectorDeleteCommand() *cobra.Command {
	var clusterName, name string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a running connector",
		Example:          `connector delete --clusterName="" --name="connector_name"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			if err := client.DeleteConnector(clusterName, name); err != nil {
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to delete, connector '%s:%s' does not exist", clusterName, name)
				return err
			}

			return bite.PrintInfo(cmd, "Connector %s:%s deleted", clusterName, name)
		},
	}

	cmd.Flags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	cmd.Flags().StringVar(&name, "name", "", `--name="connector_name"`)
	bite.CanBeSilent(cmd)

	return cmd
}
