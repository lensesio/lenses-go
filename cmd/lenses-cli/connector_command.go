package main

import (
	"fmt"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newConnectorsCommand())
	rootCmd.AddCommand(newConnectorGroupCommand())
}

func newConnectorsCommand() *cobra.Command {
	var (
		clusterName string

		namesOnly bool // if true then print only the connector names and not the details as json.
	)

	root := cobra.Command{
		Use:              "connectors",
		Short:            "List of active connectors' names",
		Aliases:          []string{"connect"},
		Example:          exampleString(`connectors or connectors --clusterName="cluster_name"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			connectorNames := make(map[string][]string) // clusterName:[] connectors names.

			if clusterName == "*" {
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
					return err
				}

				connectorNames[clusterName] = names
			}

			if namesOnly {
				var names []string
				for _, cNames := range connectorNames {
					names = append(names, cNames...)
				}

				return printJSON(cmd.OutOrStdout(), outlineStringResults("name", names))
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

			if err := printJSON(cmd.OutOrStdout(), connectors); err != nil {
				return err
			}

			return nil
		},
	}

	root.Flags().BoolVar(&namesOnly, "names", false, `--names`)

	// shared flags.
	root.PersistentFlags().StringVar(&clusterName, "clusterName", "*", `--clusterName="cluster_name"`)
	root.PersistentFlags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")
	// plugins subcommand.
	root.AddCommand(&cobra.Command{
		Use:           "plugins",
		Short:         "List of available connectors' plugins",
		Example:       exampleString(`connectors plugins --clusterName="cluster_name"`),
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

			// var b strings.Builder

			// for i, p := range plugins {
			// 	if p.Version == "null" || p.Version == "" {
			// 		p.Version = "X.X.X"
			// 	}

			// 	b.WriteString(fmt.Sprintf("Class name: %s, Type: %s, Version: %s", p.Class, p.Type, p.Version))

			// 	if !noNewLine && len(plugins)-1 != i {
			// 		// add new line if enabled and not last, note that we use the fmt.Println below
			// 		// even if newLine is disabled (for unix terminals mostly).
			// 		b.WriteString("\n")
			// 	}
			// }

			// _, err := fmt.Fprintln(cmd.OutOrStdout(), b.String())
			// return err

			for _, p := range plugins {
				if p.Version == "null" || p.Version == "" {
					p.Version = "X.X.X"
				}
			}

			return printJSON(cmd.OutOrStdout(), plugins)
		},
	})

	// clusters subcommand.

	root.AddCommand(newGetConnectorsClustersCommand())

	return &root
}

func newGetConnectorsClustersCommand() *cobra.Command {
	var namesOnly bool
	cmd := cobra.Command{
		Use:           "clusters",
		Short:         "List of available connectors' clusters",
		Example:       exampleString(`connectors clusters`),
		SilenceErrors: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			clusters, err := client.GetConnectClusters()
			if err != nil {
				return err
			}

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

			return printJSON(cmd.OutOrStdout(), clusters)
		},
	}

	cmd.Flags().BoolVar(&namesOnly, "names", false, `--names`)

	return &cmd

}

func newConnectorGroupCommand() *cobra.Command {
	var clusterName, name string

	root := cobra.Command{
		Use:              "connector",
		Short:            "Get information about a particular connector based on its name",
		Example:          exampleString(`connector --clusterName="cluster_name" --name="connector_name"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": clusterName, "name": name}); err != nil {
				return err
			}

			connector, err := client.GetConnector(clusterName, name)
			if err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("connector '%s:%s' does not exist", clusterName, name)
				return err
			}
			return printJSON(cmd.OutOrStdout(), connector)
		},
	}

	root.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	root.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	// shared flags.
	root.PersistentFlags().StringVar(&clusterName, "clusterName", "", `--clusterName="cluster_name"`)
	// root.MarkPersistentFlagRequired("clusterName") -> because of file loading, we must not require them.

	root.PersistentFlags().StringVar(&name, "name", "", `--name="connector_name"`)
	// root.MarkPersistentFlagRequired("name") -> because of file loading, we must not require them.

	// subcommands.
	root.AddCommand(newConnectorCreateCommand(&clusterName, &name))
	root.AddCommand(newConnectorUpdateCommand(&clusterName, &name))
	root.AddCommand(newConnectorGetConfigCommand(&clusterName, &name))
	root.AddCommand(newConnectorGetStatusCommand(&clusterName, &name))
	root.AddCommand(newConnectorPauseCommand(&clusterName, &name))
	root.AddCommand(newConnectorResumeCommand(&clusterName, &name))
	root.AddCommand(newConnectorRestartCommand(&clusterName, &name))
	root.AddCommand(newConnectorGetTasksCommand(&clusterName, &name))
	root.AddCommand(newConnectorDeleteCommand(&clusterName, &name))
	// connector.task subcommands.
	root.AddCommand(newConnectorTaskGroupCommand(&clusterName, &name))
	return &root
}

func newConnectorCreateCommand(clusterName *string, name *string) *cobra.Command {
	var configRaw string

	cmd := cobra.Command{
		Use:              "create",
		Short:            "Create a new connector",
		Example:          exampleString(`connector create --clusterName="cluster_name" --name="connector_name" --config="{\"key\": \"value\"}" or connector create ./connector.yml`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			connector := lenses.CreateUpdateConnectorPayload{
				ClusterAlias: *clusterName,
				Name:         *name,
				Config:       make(lenses.ConnectorConfig),
			}

			if len(args) > 0 {
				// load from file.
				if err := loadFile(cmd, args[0], &connector); err != nil {
					return err
				}
			} else {
				// try load only the config from flag or file if possible.
				if err := tryReadFile(configRaw, &connector.Config); err != nil {
					return err
				}
			}

			if err := connector.ApplyAndValidateName(); err != nil {
				return err
			}

			if err := checkRequiredFlags(cmd, flags{"clusterName": connector.ClusterAlias, "name": connector.Name}); err != nil {
				return err
			}

			_, err := client.CreateConnector(connector.ClusterAlias, connector.Name, connector.Config)
			if err != nil {
				return err
			}

			if silent {
				return nil
			}

			return echo(cmd, "Connector %s created", connector.Name)
		},
	}

	cmd.Flags().StringVar(&configRaw, "config", "", `--config="{\"key\": \"value\"}"`)

	return &cmd
}

func newConnectorUpdateCommand(clusterName *string, name *string) *cobra.Command { // almost the same as `newConnectorCreateCommand` but keep them separate, in future this may change.
	var configRaw string

	cmd := cobra.Command{
		Use:              "update",
		Short:            "Update a connector's configuration",
		Example:          exampleString(`connector update --clusterName="cluster_name" --name="connector_name" --config="{\"key\": \"value\"}" or connector update ./connector.yml`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			connector := lenses.CreateUpdateConnectorPayload{
				ClusterAlias: *clusterName,
				Name:         *name,
				Config:       make(lenses.ConnectorConfig),
			}

			if len(args) > 0 {
				// load from file.
				if err := loadFile(cmd, args[0], &connector); err != nil {
					return err
				}
			} else {
				// try load only the config from flag or file if possible.
				if err := tryReadFile(configRaw, &connector.Config); err != nil {
					return err
				}
			}

			if err := connector.ApplyAndValidateName(); err != nil {
				return err
			}

			if err := checkRequiredFlags(cmd, flags{"clusterName": connector.ClusterAlias, "name": connector.Name}); err != nil {
				return err
			}

			// for any case.
			existingConnector, err := client.GetConnector(connector.ClusterAlias, connector.Name)
			if err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("connector '%s:%s' does not exist", clusterName, name)
				return err
			}

			if existingConnector.Config != nil {
				if existingNameValue := existingConnector.Config["name"]; existingNameValue != connector.Name {
					return fmt.Errorf(`connector config["name"] '%s' does not match with the existing one '%s'`, connector.Name, existingNameValue)
				}
			}

			updatedConnector, err := client.UpdateConnector(connector.ClusterAlias, connector.Name, connector.Config)
			if err != nil {
				return err
			}

			if silent {
				return nil
			}

			echo(cmd, "Connector %s updated\n\n", connector.Name)

			return printJSON(cmd.OutOrStdout(), updatedConnector) // why we print it back? Because of the connector.Tasks.
		},
	}

	cmd.Flags().StringVar(&configRaw, "config", "", `--config="{\"key\": \"value\"}"`)
	cmd.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	cmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	return &cmd
}

func newConnectorGetConfigCommand(clusterName *string, name *string) *cobra.Command {
	cmd := cobra.Command{
		Use:              "config",
		Short:            "Get connector config",
		Example:          exampleString(`connector config --clusterName="cluster_name" --name="connector_name"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": *clusterName, "name": *name}); err != nil {
				return err
			}

			cfg, err := client.GetConnectorConfig(*clusterName, *name)
			if err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to retrieve config, connector '%s:%s' does not exist", *clusterName, *name)
				return err
			}

			return printJSON(cmd.OutOrStdout(), cfg)
		},
	}

	cmd.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	cmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	return &cmd
}

func newConnectorGetStatusCommand(clusterName *string, name *string) *cobra.Command {
	cmd := cobra.Command{
		Use:              "status",
		Short:            "Get connector status",
		Example:          exampleString(`connector status --clusterName="cluster_name" --name="connector_name"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": *clusterName, "name": *name}); err != nil {
				return err
			}

			cs, err := client.GetConnectorStatus(*clusterName, *name)
			if err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to retrieve status, connector '%s:%s' does not exist", *clusterName, *name)
				return err
			}

			return printJSON(cmd.OutOrStdout(), cs)
		},
	}

	cmd.Flags().BoolVar(&noPretty, "no-pretty", noPretty, "--no-pretty")
	cmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")
	return &cmd
}

func newConnectorPauseCommand(clusterName *string, name *string) *cobra.Command {

	cmd := cobra.Command{
		Use:              "pause",
		Short:            "Pause a connector",
		Example:          exampleString(`connector pause --clusterName="cluster_name" --name="connector_name"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": *clusterName, "name": *name}); err != nil {
				return err
			}

			if err := client.PauseConnector(*clusterName, *name); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to pause, connector '%s:%s' does not exist", *clusterName, *name)
				return err
			}

			return echo(cmd, "Connector %s:%s paused", *clusterName, *name)
		},
	}

	return &cmd
}

func newConnectorResumeCommand(clusterName *string, name *string) *cobra.Command {

	cmd := cobra.Command{
		Use:              "resume",
		Short:            "Resume a paused connector",
		Example:          exampleString(`connector resume --clusterName="cluster_name" --name="connector_name"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": *clusterName, "name": *name}); err != nil {
				return err
			}

			if err := client.ResumeConnector(*clusterName, *name); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to resume, connector '%s:%s' does not exist", *clusterName, *name)
				return err
			}

			return echo(cmd, "Connector %s:%s resumed", *clusterName, *name)
		},
	}

	return &cmd
}

func newConnectorRestartCommand(clusterName *string, name *string) *cobra.Command {

	cmd := cobra.Command{
		Use:              "restart",
		Short:            "Restart a connector",
		Example:          exampleString(`connector restart --clusterName="cluster_name" --name="connector_name"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": *clusterName, "name": *name}); err != nil {
				return err
			}

			if err := client.RestartConnector(*clusterName, *name); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to restart, connector '%s:%s' does not exist", *clusterName, *name)
				return err
			}

			return echo(cmd, "Connector %s:%s restarted", *clusterName, *name)
		},
	}

	return &cmd
}

func newConnectorGetTasksCommand(clusterName *string, name *string) *cobra.Command {
	cmd := cobra.Command{
		Use:              "tasks",
		Short:            "List of connector tasks",
		Example:          exampleString(`connector tasks --clusterName="cluster_name" --name="connector_name"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": *clusterName, "name": *name}); err != nil {
				return err
			}

			tasksMap, err := client.GetConnectorTasks(*clusterName, *name)
			if err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to retrieve tasks, connector '%s:%s' does not exist", *clusterName, *name)
				return err
			}

			return printJSON(cmd.OutOrStdout(), tasksMap)
		},
	}

	cmd.Flags().BoolVar(&noPretty, "no-pretty", noPretty, `--no-pretty`)
	cmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	return &cmd
}

func newConnectorTaskGroupCommand(clusterName *string, name *string) *cobra.Command {
	rootSub := cobra.Command{
		Use:              "task",
		Short:            "Work with a particular connector task, see connector task --help for details",
		Example:          exampleString(`connector task status --clusterName="cluster_name" --name="connector_name" --task=1`),
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	taskID := rootSub.PersistentFlags().Int("task", 0, "--task=1 The Task ID")
	rootSub.MarkPersistentFlagRequired("task")

	rootSub.AddCommand(newConnectorGetCurrentTaskStatusCommand(clusterName, name, taskID))
	rootSub.AddCommand(newConnectorTaskRestartCommand(clusterName, name, taskID))

	return &rootSub
}

func newConnectorGetCurrentTaskStatusCommand(clusterName *string, name *string, taskID *int) *cobra.Command {
	cmd := cobra.Command{
		Use:              "status",
		Short:            "Get current status of a task",
		Example:          exampleString(`connector task status --clusterName="cluster_name" --name="connector_name" --task=1`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": *clusterName, "name": *name}); err != nil {
				return err
			}

			cst, err := client.GetConnectorTaskStatus(*clusterName, *name, *taskID)
			if err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("task does not exist")
				return err
			}

			return printJSON(cmd.OutOrStdout(), cst)
		},
	}

	cmd.Flags().BoolVar(&noPretty, "no-pretty", noPretty, `--no-pretty`)
	cmd.Flags().StringVarP(&jmespathQuery, "query", "q", "", "jmespath query to further filter results")

	return &cmd
}

func newConnectorTaskRestartCommand(clusterName *string, name *string, taskID *int) *cobra.Command {

	cmd := cobra.Command{
		Use:              "restart",
		Short:            "Restart a connector task",
		Example:          exampleString(`connector task restart --clusterName="cluster_name" --name="connector_name" --task=1`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": *clusterName, "name": *name}); err != nil {
				return err
			}

			if err := client.RestartConnectorTask(*clusterName, *name, *taskID); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("task does not exist")
				return err
			}

			return echo(cmd, "Connector task %s:%s:%d restarted", *clusterName, *name, *taskID)
		},
	}

	return &cmd
}

func newConnectorDeleteCommand(clusterName *string, name *string) *cobra.Command {

	cmd := cobra.Command{
		Use:              "delete",
		Short:            "Delete a running connector",
		Example:          exampleString(`connector delete --clusterName="" --name="connector_name"`),
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"clusterName": *clusterName, "name": *name}); err != nil {
				return err
			}

			if err := client.DeleteConnector(*clusterName, *name); err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to delete, connector '%s:%s' does not exist", *clusterName, *name)
				return err
			}

			return echo(cmd, "Connector %s:%s deleted", *clusterName, *name)
		},
	}

	return &cmd
}
