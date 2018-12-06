package main

import (
	"github.com/kataras/golog"
	"fmt"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newUserGroupCommand())
}

func newUserGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "user",
		Short:            "List information about the authenticated logged user such as the given roles given by the lenses administrator or manage the user's profile",
		Example:          "user",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if user := client.User; user.Name != "" {
				// if logged in using the user password, then we have those info,
				// let's print it as well.
				return bite.PrintObject(cmd, user)
			}
			return nil
		},
	}

	bite.CanPrintJSON(root)

	root.AddCommand(newUserProfileGroupCommand())

	return root
}

func newUserProfileGroupCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "profile",
		Short:            "List the user-specific favourites, if any",
		Example:          "user profile",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := client.GetUserProfile()
			if err != nil {
				return err
			}

			if bite.ExpectsFeedback(cmd) && len(profile.Topics)+len(profile.Schemas)+len(profile.Transformers) == 0 {
				// do not throw error, it's not an error.
				return bite.PrintInfo(cmd, "No user profile available.")
			}

			return bite.PrintObject(cmd, profile)
		},
	}

	bite.CanBeSilent(rootSub)
	bite.CanPrintJSON(rootSub)

	rootSub.AddCommand(newCreateUserProfilePropertyValueCommand())
	rootSub.AddCommand(newDeleteUserProfilePropertyValueCommand())

	return rootSub
}

func walkPropertyValueFromArgs(args []string, actionFunc func(property, value string) error) error {
	if len(args) < 2 {
		return fmt.Errorf("at least two arguments are required, the first is the property name and the second is the actual property's value")
	}

	for i, n := 0, len(args); i < n; i++ {
		property := args[i]
		i++
		if i >= n {
			break
		}
		value := args[i]

		if err := actionFunc(property, value); err != nil {
			return err
		}
	}

	return nil
}

func newCreateUserProfilePropertyValueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"add", "insert", "create", "update"},
		Short:            `Add a "value" to the user profile "property" entries`,
		Example:          "user profile set newProperty newValueToTheProperty",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {

			return walkPropertyValueFromArgs(args, func(property, value string) error {
				if err := client.CreateUserProfilePropertyValue(property, value); err != nil {
					golog.Errorf("Failed to add the user profile value [%s] from property [%s]. [%s]", value, property, err.Error())
					return err
				}

				return bite.PrintInfo(cmd, "User profile value: [%s] for property: [%s] added", value, property)
			})
		},
	}

	bite.CanBeSilent(cmd)

	return cmd
}

func newDeleteUserProfilePropertyValueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "delete",
		Aliases:          []string{"remove"},
		Short:            `Remove the "value" from the user profile "property" entries`,
		Example:          "user profile delete existingProperty existingValueFromProperty",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return walkPropertyValueFromArgs(args, func(property, value string) error {
				if err := client.DeleteUserProfilePropertyValue(property, value); err != nil {
					golog.Errorf("Failed to remove the user profile value [%s] from property [%s]. [%s]", value, property, err.Error())
					return err
				}

				return bite.PrintInfo(cmd, "User profile value: [%s] from property: [%s] removed", value, property)
			})
		},
	}

	bite.CanBeSilent(cmd)

	return cmd
}
