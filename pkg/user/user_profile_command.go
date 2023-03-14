package user

import (
	"github.com/kataras/golog"

	"github.com/lensesio/bite"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewUserGroupCommand creates `user` command
func NewUserGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "user",
		Short:            "List information about the authenticated logged user such as the given roles given by the lenses administrator or manage the user's profile",
		Example:          "user",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if user := config.Client.User; user.Name != "" {
				// if logged in using the user password, then we have those info,
				// let's print it as well.
				return bite.PrintObject(cmd, user)
			}
			return nil
		},
	}

	bite.CanPrintJSON(root)

	root.AddCommand(NewUserProfileGroupCommand())

	return root
}

// NewUserProfileGroupCommand creates `users profile` command
func NewUserProfileGroupCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "profile",
		Short:            "List the user-specific favourites, if any",
		Example:          "user profile",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			profile, err := config.Client.GetUserProfile()
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

	rootSub.AddCommand(NewCreateUserProfilePropertyValueCommand())
	rootSub.AddCommand(NewDeleteUserProfilePropertyValueCommand())

	return rootSub
}

// NewCreateUserProfilePropertyValueCommand creates `profile user set` command
func NewCreateUserProfilePropertyValueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"add", "insert", "create", "update"},
		Short:            `Add a "value" to the user profile "property" entries`,
		Example:          "user profile set newProperty newValueToTheProperty",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {

			return utils.WalkPropertyValueFromArgs(args, func(property, value string) error {
				if err := config.Client.CreateUserProfilePropertyValue(property, value); err != nil {
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

// NewDeleteUserProfilePropertyValueCommand creates `profile user delete` command
func NewDeleteUserProfilePropertyValueCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "delete",
		Aliases:          []string{"remove"},
		Short:            `Remove the "value" from the user profile "property" entries`,
		Example:          "user profile delete existingProperty existingValueFromProperty",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return utils.WalkPropertyValueFromArgs(args, func(property, value string) error {
				if err := config.Client.DeleteUserProfilePropertyValue(property, value); err != nil {
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
