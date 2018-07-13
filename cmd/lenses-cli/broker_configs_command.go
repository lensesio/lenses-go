package main

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	// Hidden: true
	// app.AddCommand(newDynamicClusterConfigsGroupCommand())
	// app.AddCommand(newDynamicBrokerConfigsGroupCommand())
}

func newDynamicClusterConfigsGroupCommand() *cobra.Command {
	root := &cobra.Command{
		// currently we don't have an API to retrieve all brokers' ids(...),
		// and no cluster "static" configs,
		// so the only thing it will print is the dynamic updated configs for all brokers for a kafka cluster.
		Use:              "cluster",
		Short:            "Work with the dynamic updated configurations for a kafka cluster",
		Example:          `cluster configs`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	root.AddCommand(newGetDynamicClusterConfigsCommand())

	return root
}

func newGetDynamicClusterConfigsCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "configs",
		Short:            "List the dynamic updated configurations for a kafka cluster",
		Example:          `cluster configs`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			configs, err := client.GetDynamicClusterConfigs()
			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, configs)
		},
	}

	rootSub.AddCommand(newSetDynamicClusterConfigsCommand())
	rootSub.AddCommand(newDeleteDynamicClusterConfigsCommand())

	return rootSub
}

func newSetDynamicClusterConfigsCommand() *cobra.Command {
	var (
		configsRaw string
		configs    lenses.BrokerConfig
	)

	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"add", "update"},
		Short:            "Add or update configuration for a kafka cluster dynamically",
		Example:          `cluster configs set --configs=file.yml/json or --configs="{\"log.cleaner.threads\": 2, \"compression.type\": \"snappy\"}"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bite.FriendlyError(cmd, errResourceInternal, "failed to retrieve cluster configurations")
			bite.FriendlyError(cmd, errResourceNotGoodMessage, "unknown configurations where provided: %#+v", configs)

			if err := bite.TryReadFile(configsRaw, &configs); err != nil {
				// from flag as json.
				if err = json.Unmarshal([]byte(configsRaw), &configs); err != nil {
					return fmt.Errorf("unable to unmarshal the configs: %v. Try using a yaml or json file instead", err)
				}
			}

			err := client.UpdateDynamicClusterConfigs(configs)
			if err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Cluster configs updated.")
		},
	}

	cmd.Flags().StringVar(&configsRaw, "configs", "", `--configs="{\"log.cleaner.threads\": 2, \"compression.type\": \"snappy\"}`)
	cmd.MarkFlagRequired("configs")

	bite.CanBeSilent(cmd)
	return cmd
}

func newDeleteDynamicClusterConfigsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "delete",
		Aliases:          []string{"reset"},
		Short:            "Delete/reset cluster configuration dynamically, separate them by space",
		Example:          `cluster configs delete log.cleaner.threads compression.type snappy`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, keysToBeReseted []string) error {
			if len(keysToBeReseted) == 0 {
				return bite.PrintInfo(cmd, "keys are required, pass the config's keys to be removed/reset to their default through command's arguments separated by space")
			}

			bite.FriendlyError(cmd, errResourceInternal, "failed to retrieve cluster configurations")
			keysStr := strings.Join(keysToBeReseted, ", ")
			bite.FriendlyError(cmd, errResourceNotGoodMessage, "unknown keys where provided: %s", keysStr)

			err := client.DeleteDynamicClusterConfigs(keysToBeReseted...)
			if err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Cluster configs %s reseted.", keysStr)
		},
	}

	bite.CanBeSilent(cmd)
	return cmd
}

func newDynamicBrokerConfigsGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "broker",
		Short:            "Work with broker configurations",
		Example:          `broker configs --broker=brokerID`,
		Hidden:           true,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	root.AddCommand(newGetDynamicBrokerConfigsCommand())

	return root
}

func newGetDynamicBrokerConfigsCommand() *cobra.Command {
	var brokerID int
	rootSub := &cobra.Command{
		Use:              "configs",
		Short:            "List the dynamic updated configurations for a kafka broker",
		Example:          `broker configs --broker=brokerID`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bite.FriendlyError(cmd, errResourceInternal, "could not retrieve configurations for broker: %d", brokerID)
			bite.FriendlyError(cmd, errResourceNotFoundMessage, "could not retrieve broker: %d", brokerID)

			configs, err := client.GetDynamicBrokerConfigs(brokerID)
			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, configs)
		},
	}

	rootSub.Flags().IntVar(&brokerID, "broker", 0, "--broker=42")
	rootSub.MarkFlagRequired("broker")

	rootSub.AddCommand(newSetDynamicBrokerConfigsCommand())
	rootSub.AddCommand(newDeleteDynamicBrokerConfigsCommand())
	return rootSub
}

func newSetDynamicBrokerConfigsCommand() *cobra.Command {
	var (
		brokerID   int
		configsRaw string
		configs    lenses.BrokerConfig
	)

	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"add", "update"},
		Short:            "Add or update broker configuration dynamically",
		Example:          `broker configs set --broker=brokerID --configs=file.yml/json or --configs="{\"log.cleaner.threads\": 2, \"compression.type\": \"snappy\"}"`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bite.FriendlyError(cmd, errResourceInternal, "failed to retrieve broker configurations", brokerID)
			bite.FriendlyError(cmd, errResourceNotGoodMessage, "unknown configurations where provided: %#+v", configs)

			if err := bite.TryReadFile(configsRaw, &configs); err != nil {
				// from flag as json.
				if err = json.Unmarshal([]byte(configsRaw), &configs); err != nil {
					return fmt.Errorf("unable to unmarshal the configs: %v. Try using a yaml or json file instead", err)
				}
			}

			err := client.UpdateDynamicBrokerConfigs(brokerID, configs)
			if err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Configs updated for broker with id: %d.", brokerID)
		},
	}

	cmd.Flags().IntVar(&brokerID, "broker", 0, "--broker=brokerID")
	cmd.MarkFlagRequired("broker")

	cmd.Flags().StringVar(&configsRaw, "configs", "", `--configs="{\"log.cleaner.threads\": 2, \"compression.type\": \"snappy\"}`)
	cmd.MarkFlagRequired("configs")

	bite.CanBeSilent(cmd)
	return cmd
}

func newDeleteDynamicBrokerConfigsCommand() *cobra.Command {
	var brokerID int

	cmd := &cobra.Command{
		Use:              "delete",
		Aliases:          []string{"reset"},
		Short:            "Delete/reset broker configuration dynamically, separate them by space",
		Example:          `broker configs delete --broker=brokerID log.cleaner.threads compression.type snappy`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, keysToBeReseted []string) error {
			if len(keysToBeReseted) == 0 {
				return bite.PrintInfo(cmd, "keys are required, pass the config's keys to be removed/reseted to their default values through command's arguments separated by space")
			}

			bite.FriendlyError(cmd, errResourceInternal, "could not retrieve configurations for broker: %d", brokerID)
			keysStr := strings.Join(keysToBeReseted, ", ")
			bite.FriendlyError(cmd, errResourceNotGoodMessage, "unknown keys where provided: %s", keysStr)

			err := client.DeleteDynamicBrokerConfigs(brokerID, keysToBeReseted...)
			if err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Configs %s reseted for broker with id: %d.", keysStr, brokerID)
		},
	}

	cmd.Flags().IntVar(&brokerID, "broker", 0, "--broker=brokerID")
	cmd.MarkFlagRequired("broker")

	bite.CanBeSilent(cmd)
	return cmd
}
