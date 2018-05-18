package main

import (
	"fmt"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newGetQuotasCommand())
	rootCmd.AddCommand(newQuotaGroupCommand())
}

func newGetQuotasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "quotas",
		Short:            "List of all available quotas",
		Example:          exampleString("quotas"),
		TraverseChildren: true,
	}

	shouldPrintJSON(cmd, func() (interface{}, error) {
		return client.GetQuotas()
	})

	return cmd
}

func newQuotaGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "quota",
		Short:            "Work with particular a quota, create a new quota or update and delete an existing one",
		Example:          exampleString(`quota users [--quota-user=""] [--quota-client=""] --quota-config="{\"producer_byte_rate\": \"100000\"},\"consumer_byte_rate\": \"200000\"},\"request_percentage\": \"75\"}"`),
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	root.AddCommand(newQuotaUsersSubGroupCommand())
	root.AddCommand(newQuotaClientsSubGroupCommand())

	return root
}

type createQuotaPayload struct {
	Config lenses.QuotaConfig `yaml:"Config"`
	// for specific user and/or client.
	User string `yaml:"User"`
	// if "all" or "*" then means all clients.
	// Minor note On quota clients set/create/update the Config and Client field are used only.
	ClientID string `yaml:"Client"`
}

func newQuotaUsersSubGroupCommand() *cobra.Command {
	var (
		configRaw string
		quota     createQuotaPayload
	)

	rootSub := &cobra.Command{
		Use:              "users",
		Short:            "Work with users quotas",
		Example:          exampleString("users"),
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	setCommand := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or update quota for all users or for a specific user (and client)",
		Example:          exampleString(`quota users set --quota-user="user" --quota-client="" --quota-config="{\"producer_byte_rate\": \"100000\"},\"consumer_byte_rate\": \"200000\"},\"request_percentage\": \"75\"}"`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"quota-config": configRaw}); err != nil {
				return err
			}

			errResourceNotAccessibleMessage = "unable to access quota, user has no rights for this action"

			if quota.User != "" {
				if clientID := quota.ClientID; clientID != "" {
					if clientID == "all" || clientID == "*" {
						if err := client.CreateOrUpdateQuotaForUserAllClients(quota.User, quota.Config); err != nil {
							return err
						}

						return echo(cmd, "Quota for user %s and all clients set", quota.User)

					}

					if err := client.CreateOrUpdateQuotaForUserClient(quota.User, clientID, quota.Config); err != nil {
						return err
					}

					return echo(cmd, "Quota for user %s and client %s set", quota.User, clientID)
				}

				if err := client.CreateOrUpdateQuotaForUser(quota.User, quota.Config); err != nil {
					return err
				}

				return echo(cmd, "Quota for user %s created", quota.User)
			}

			if err := client.CreateOrUpdateQuotaForAllUsers(quota.Config); err != nil {
				return err
			}

			return echo(cmd, "Quota for all users created")
		},
	}

	setCommand.Flags().StringVar(&configRaw, "quota-config", "", `--quota-config="{\"key\": \"value\"}"`)
	setCommand.Flags().StringVar(&quota.User, "quota-user", "", "--quota-user=")
	setCommand.Flags().StringVar(&quota.ClientID, "quota-client", "", "--quota-client=")

	shouldTryLoadFile(setCommand, &quota).Else(func() error { return tryReadFile(configRaw, &quota.Config) })

	rootSub.AddCommand(setCommand)

	deleteCommand := &cobra.Command{
		Use:              "delete",
		Short:            "Delete default quota for all users or for a specific user (and client)",
		Example:          exampleString(`quota users delete [to delete for all users] or --quota-client=* [for all clients or to a specific one] or --quota-user="user"`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			errResourceNotAccessibleMessage = "unable to delete quota, user has no rights for this action"

			var user, clientID = quota.User, quota.ClientID

			if user != "" {
				if clientID != "" {
					if clientID == "all" || clientID == "*" {
						if err := client.DeleteQuotaForUserAllClients(user); err != nil {
							errResourceNotFoundMessage = fmt.Sprintf("unable to delete, quota for user: '%s' does not exist", user)
							return err
						}

						return echo(cmd, "Quota for user %s deleted for all clients", user)
					}

					if err := client.DeleteQuotaForUserClient(user, clientID); err != nil {
						errResourceNotFoundMessage = fmt.Sprintf("unable to delete, quota for user: '%s' and client: '%s' does not exist", user, clientID)
						return err
					}

					return echo(cmd, "Quota for user %s deleted for client %s", user, clientID)
				}

				if err := client.DeleteQuotaForUser(user); err != nil {
					errResourceNotFoundMessage = fmt.Sprintf("unable to delete, quota for user: '%s' does not exist", user)
					return err
				}

				return echo(cmd, "Quota for user %s deleted", user)
			}

			if err := client.DeleteQuotaForAllUsers(); err != nil {
				return err
			}

			return echo(cmd, "Quota for all users deleted")
		},
	}

	deleteCommand.Flags().StringVar(&quota.User, "quota-user", "", "--quota-user=")
	deleteCommand.Flags().StringVar(&quota.ClientID, "quota-client", "", "--quota-client=")
	rootSub.AddCommand(deleteCommand)

	return rootSub
}

func newQuotaClientsSubGroupCommand() *cobra.Command {
	var (
		configRaw string
		quota     createQuotaPayload
	)

	rootSub := &cobra.Command{
		Use:              "clients",
		Short:            "Work with clients quotas",
		Example:          exampleString("clients"),
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	setCommand := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or update quota for all clients or for a specific one",
		Example:          exampleString(`lenses-cli quota users set --quota-user="user" --quota-client="" --quota-config="{\"producer_byte_rate\": \"100000\"},\"consumer_byte_rate\": \"200000\"},\"request_percentage\": \"75\"}"`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"quota-config": configRaw}); err != nil {
				return err
			}

			errResourceNotAccessibleMessage = "unable to access quota, user has no rights for this action"

			if id := quota.ClientID; id != "" && id != "all" && id != "*" {
				if err := client.CreateOrUpdateQuotaForClient(quota.ClientID, quota.Config); err != nil {
					return err
				}

				return echo(cmd, "Quota for client %s created", quota.ClientID)
			}

			if err := client.CreateOrUpdateQuotaForAllClients(quota.Config); err != nil {
				return err
			}

			return echo(cmd, "Quota for all clients created")
		},
	}

	setCommand.Flags().StringVar(&configRaw, "quota-config", "", `--quota-config="{\"key\": \"value\"}"`)
	setCommand.Flags().StringVar(&quota.ClientID, "quota-client", "", "--quota-client=")

	shouldTryLoadFile(setCommand, &quota).Else(func() error { return tryReadFile(configRaw, &quota.Config) })

	rootSub.AddCommand(setCommand)

	deleteCommand := &cobra.Command{
		Use:              "delete",
		Short:            "Delete default quota for all clients or for a specific one",
		Example:          exampleString(`quota users delete [--quota-user=""] [--quota-client=""]`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"quota-client": quota.ClientID}); err != nil {
				return err
			}

			errResourceNotAccessibleMessage = "unable to delete quota, user has no rights for this action"

			if id := quota.ClientID; id != "" && id != "all" && id != "*" {
				if err := client.DeleteQuotaForClient(id); err != nil {
					errResourceNotFoundMessage = fmt.Sprintf("unable to delete, quota for client: '%s' does not exist", id)
					return err
				}

				return echo(cmd, "Quota for client %s deleted", id)
			}

			if err := client.DeleteQuotaForAllClients(); err != nil {
				return err
			}

			return echo(cmd, "Quota for all clients deleted")
		},
	}

	deleteCommand.Flags().StringVar(&quota.ClientID, "quota-client", "", "--quota-client=")

	rootSub.AddCommand(deleteCommand)

	return rootSub
}
