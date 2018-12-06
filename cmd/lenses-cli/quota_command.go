package main

import (
	"github.com/kataras/golog"
	"strings"

	"github.com/landoop/lenses-go"
	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newGetQuotasCommand())
	app.AddCommand(newQuotaGroupCommand())
}

func newGetQuotasCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "quotas",
		Short:            "List of all available quotas",
		Example:          "quotas",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			quotas, err := client.GetQuotas()
			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, quotas)
		},
	}

	bite.CanPrintJSON(cmd)

	return cmd
}

func newQuotaGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "quota",
		Short:            "Manage a quota, create a new quota or update and delete an existing one",
		Example:          `quota users set [--quota-user=""] [--quota-client=""] --quota-config="{\"producer_byte_rate\": \"100000\",\"consumer_byte_rate\": \"200000\",\"request_percentage\": \"75\"}"`,
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	root.AddCommand(newQuotaUsersSubGroupCommand())
	root.AddCommand(newQuotaClientsSubGroupCommand())

	return root
}

func newQuotaUsersSubGroupCommand() *cobra.Command {
	var (
		configRaw string
		quotas    []lenses.CreateQuotaPayload
		quota     lenses.CreateQuotaPayload
	)

	rootSub := &cobra.Command{
		Use:              "users",
		Short:            "Manage users quotas",
		Example:          "quota users",
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	setCommand := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or update the default user quota or a specific user quota (and/or client(s))",
		Example:          `quota users set [--quota-user="user"] [--quota-client=""] --quota-config="{\"producer_byte_rate\": \"100000\",\"consumer_byte_rate\": \"200000\",\"request_percentage\": \"75\"}"`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(quotas) > 0 {
				for _, quota := range quotas {
					err := CreateQuotaForUsers(cmd, quota)
					if err != nil {
						return err
					}
				}

				return nil
			}

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"quota-config": configRaw}); err != nil {
				return err
			}

			err := CreateQuotaForUsers(cmd, quota)
			if err != nil {
				golog.Errorf("Failed to create quota for user [%s], client [%s]. [%s]", quota.User, quota.ClientID, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Default user quota created/updated")
		},
	}

	setCommand.Flags().StringVar(&configRaw, "quota-config", "", `Quota config .e.g. "{\"key\": \"value\"}"`)
	setCommand.Flags().StringVar(&quota.User, "quota-user", "", "Quota user")
	setCommand.Flags().StringVar(&quota.ClientID, "quota-client", "", "Quota client")

	bite.CanBeSilent(setCommand)
	bite.Prepend(setCommand, bite.FileBind(&quotas, bite.ElseBind(func() error { return bite.TryReadFile(configRaw, &quota.Config) })))

	rootSub.AddCommand(setCommand)

	deleteCommand := &cobra.Command{
		Use:              "delete",
		Short:            "Delete the default user quota or a specific quota for a user (and client)",
		Example:          `quota users delete [to delete for all users] or --quota-client=* [for all clients or to a specific one] or --quota-user="user" producer_byte_rate or/and consumer_byte_rate or/and request_percentage to remove quota's config properties, if arguments empty then all keys will be passed on`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			actionMsg := "delete" // +d in the echo message.
			if len(args) > 0 {
				// if arguments are not empty then it should show "update(d)",
				// otherwise it's a deletion because it deletes the whole default user quota.
				actionMsg = "update"
			}

			// bite.FriendlyError(cmd, errResourceNotAccessibleMessage, "unable to %s quota, user has no rights for this action", actionMsg)

			var user, clientID = quota.User, quota.ClientID

			if user != "" {
				if clientID != "" {
					if clientID == "all" || clientID == "*" {
						if err := client.DeleteQuotaForUserAllClients(user, args...); err != nil {
							bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to [%s], quota for user: [%s] does not exist", actionMsg, user)
							return err
						}

						return bite.PrintInfo(cmd, "Quota for user [%s] deleted for all clients", user)
					}

					if err := client.DeleteQuotaForUserClient(user, clientID, args...); err != nil {
						golog.Errorf("Failed to delete quota for user [%s], client [%s]. [%s]", quota.User, quota.ClientID, err.Error())
						return err
					}

					return bite.PrintInfo(cmd, "Quota for user [%s] deleted for client [%s]", user, clientID)
				}

				if err := client.DeleteQuotaForUser(user, args...); err != nil {
					golog.Errorf("Failed to delete quota for user [%s], client [%s]. [%s]", quota.User, quota.ClientID, err.Error())
					return err
				}

				return bite.PrintInfo(cmd, "Quota for user [%s] [%sd]", user, actionMsg)
			}

			if err := client.DeleteQuotaForAllUsers(args...); err != nil {
				golog.Errorf("Failed to create quota for all users. [%s]", err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Default user quota [%sd]", actionMsg)
		},
	}

	deleteCommand.Flags().StringVar(&quota.User, "quota-user", "", "Quota user")
	deleteCommand.Flags().StringVar(&quota.ClientID, "quota-client", "", "Quota client")
	bite.CanBeSilent(deleteCommand)

	rootSub.AddCommand(deleteCommand)

	return rootSub
}

func newQuotaClientsSubGroupCommand() *cobra.Command {
	var (
		configRaw string
		quota     lenses.CreateQuotaPayload
		quotas    []lenses.CreateQuotaPayload
	)

	rootSub := &cobra.Command{
		Use:              "clients",
		Short:            "Manage clients quotas",
		Example:          "quota clients",
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	setCommand := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or update the default client quota or for a specific client",
		Example:          `quota clients set [--quota-client=""] --quota-config="{\"producer_byte_rate\": \"100000\",\"consumer_byte_rate\": \"200000\",\"request_percentage\": \"75\"}"`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(quotas) > 0 {
				for _, quota := range quotas {
					err := CreateQuotaForClients(cmd, quota)
					if err != nil {
						return err
					}
				}
				return nil

			}

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"quota-config": configRaw}); err != nil {
				return err
			}

			err := CreateQuotaForClients(cmd, quota)

			if err != nil {
				golog.Errorf("Failed to create quota for client [%s]. [%s]", quota.ClientID, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Default client quota created/updated")
		},
	}

	setCommand.Flags().StringVar(&configRaw, "quota-config", "", `Quota config .e.g. "{\"key\": \"value\"}"`)
	setCommand.Flags().StringVar(&quota.ClientID, "quota-client", "", "Quota client")
	bite.CanBeSilent(setCommand)
	bite.Prepend(setCommand, bite.FileBind(&quotas, bite.ElseBind(func() error { return bite.TryReadFile(configRaw, &quota.Config) })))

	rootSub.AddCommand(setCommand)

	deleteCommand := &cobra.Command{
		Use:              "delete",
		Short:            "Delete the default client quota or a specific one",
		Example:          `quota clients delete [--quota-client=""] producer_byte_rate or/and consumer_byte_rate or/and request_percentage to remove quota's config properties, if arguments empty then all keys will be passed on`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			actionMsg := "delete" // +d in the echo message.
			if len(args) > 0 {
				// if arguments are not empty then it should show "update(d)",
				// otherwise it's a deletion because it deletes the whole default user quota.
				actionMsg = "update"
			}

			// bite.FriendlyError(cmd, errResourceNotAccessibleMessage, "unable to %s quota, user has no rights for this action", actionMsg)

			if id := quota.ClientID; id != "" && id != "all" && id != "*" {
				if err := client.DeleteQuotaForClient(id, args...); err != nil {
					golog.Errorf("Failed to delete quota for client [%s]. [%s]", quota.ClientID, err.Error())
					return err
				}

				return bite.PrintInfo(cmd, "Quota for client [%s] [%sd]", id, actionMsg)
			}

			if err := client.DeleteQuotaForAllClients(args...); err != nil {
				golog.Errorf("Failed to delete quota for all clients. [%s]", err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Default client quota [%sd]", actionMsg)
		},
	}

	deleteCommand.Flags().StringVar(&quota.ClientID, "quota-client", "", "Quota client")
	bite.CanBeSilent(deleteCommand)

	rootSub.AddCommand(deleteCommand)

	return rootSub
}

func CreateQuotaForClients(cmd *cobra.Command, quota lenses.CreateQuotaPayload) error {
	if id := quota.ClientID; id != "" && id != "all" && id != "*" && strings.HasPrefix(quota.QuotaType, "CLIENT") {
		if err := client.CreateOrUpdateQuotaForClient(quota.ClientID, quota.Config); err != nil {
			return err
		}

		return bite.PrintInfo(cmd, "Quota for client [%s] created/updated", quota.ClientID)
	}

	err := client.CreateOrUpdateQuotaForAllClients(quota.Config)
	return err
}

func CreateQuotaForUsers(cmd *cobra.Command, quota lenses.CreateQuotaPayload) error {
	if quota.User != "" && strings.HasPrefix(quota.QuotaType, "USER") {
		if clientID := quota.ClientID; clientID != "" {
			if clientID == "all" || clientID == "*" {
				if err := client.CreateOrUpdateQuotaForUserAllClients(quota.User, quota.Config); err != nil {
					return err
				}

				return bite.PrintInfo(cmd, "Quota for user [%s] and all clients created/updated", quota.User)

			}

			if err := client.CreateOrUpdateQuotaForUserClient(quota.User, clientID, quota.Config); err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Quota for user [%s] and client [%s] created/updated", quota.User, clientID)
		}

		if err := client.CreateOrUpdateQuotaForUser(quota.User, quota.Config); err != nil {
			return err
		}

		return bite.PrintInfo(cmd, "Quota for user [%s] created/updated", quota.User)
	}

	err := client.CreateOrUpdateQuotaForAllUsers(quota.Config)

	return err
}
