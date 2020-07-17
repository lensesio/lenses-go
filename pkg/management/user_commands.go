package management

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewUsersCommand creates the `groups` command
func NewUsersCommand() *cobra.Command {
	var groupNames []string
	root := &cobra.Command{
		Use:   "users",
		Short: "Manage users",
		Example: `
users
users --groups Group1 --groups Group2
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			users, err := config.Client.GetUsers()
			if err != nil {
				golog.Errorf("Failed to find users. [%s]", err.Error())
				return err
			}

			if len(groupNames) > 0 {
				var filteredUsers []api.UserMember
				for _, user := range users {
					var filteredUser api.UserMember
					for _, group := range groupNames {
						if utils.StringInSlice(group, user.Groups) {
							filteredUser = user
						}
					}
					if filteredUser.Username != "" {
						filteredUsers = append(filteredUsers, filteredUser)
					}
				}
				return bite.PrintObject(cmd, filteredUsers)
			}
			return bite.PrintObject(cmd, users)
		},
	}

	root.Flags().StringArrayVar(&groupNames, "groups", []string{}, `Group name`)

	root.AddCommand(NewGetUserCommand())
	root.AddCommand(NewCreateUserCommand())
	root.AddCommand(NewDeleteUserCommand())
	root.AddCommand(NewUpdateUserCommand())
	root.AddCommand(NewPasswordUserCommand())
	return root
}

//NewGetUserCommand creates `groups get`
func NewGetUserCommand() *cobra.Command {
	var userName string

	cmd := &cobra.Command{
		Use:              "get",
		Short:            "Get the user by provided name",
		Example:          "users get --username=johndoe",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			user, err := config.Client.GetUser(userName)
			if err != nil {
				return fmt.Errorf("Failed to find user. [%s]", err.Error())
			}
			return bite.PrintObject(cmd, user)
		},
	}

	cmd.Flags().StringVar(&userName, "username", "", `User username`)
	bite.CanPrintJSON(cmd)
	return cmd
}

//NewCreateUserCommand creates a new user
func NewCreateUserCommand() *cobra.Command {
	var user api.UserMember

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a user",
		Example: `
users create --username john --password secretpass --security basic --groups MyGroup
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := bite.FlagPair{
				"username": user.Username,
				"groups":   user.Groups,
				"security": user.Type,
			}
			if err := bite.CheckRequiredFlags(cmd, flags); err != nil {
				return err
			}

			if err := config.Client.CreateUser(&user); err != nil {
				return fmt.Errorf("Failed to create user [%s]. [%s]", user.Username, err.Error())
			}

			return bite.PrintInfo(cmd, "User [%s] created", user.Username)
		},
	}

	cmd.Flags().StringVar(&user.Username, "username", "", "User username")
	cmd.Flags().StringVar(&user.Password, "password", "", "User Password")
	cmd.Flags().StringVar(&user.Email, "email", "", "User email")
	cmd.Flags().StringVar(&user.Type, "security", "", "User security type")
	cmd.Flags().StringArrayVar(&user.Groups, "groups", []string{}, "User groups")

	bite.Prepend(cmd, bite.FileBind(&user))
	bite.CanBeSilent(cmd)

	return cmd
}

//NewUpdateUserCommand creates a new user
func NewUpdateUserCommand() *cobra.Command {
	var user api.UserMember

	cmd := &cobra.Command{
		Use:   "update",
		Short: "update a group",
		Example: `
users update --username john --groups MyGroup
users update --username john --email johndoe@mail.com --groups MyGroup
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := bite.FlagPair{
				"username": user.Username,
			}
			if len(user.Groups) == 0 {
				return fmt.Errorf("required flag \"groups\" not set")
			}
			if err := bite.CheckRequiredFlags(cmd, flags); err != nil {
				return err
			}

			if err := config.Client.UpdateUser(&user); err != nil {
				return fmt.Errorf("Failed to update user [%s]. [%s]", user.Username, err.Error())
			}

			return bite.PrintInfo(cmd, "User [%s] updated", user.Username)
		},
	}

	cmd.Flags().StringVar(&user.Username, "username", "", "User username")
	cmd.Flags().StringVar(&user.Email, "email", "", "User email")
	cmd.Flags().StringArrayVar(&user.Groups, "groups", []string{}, "User groups")

	bite.Prepend(cmd, bite.FileBind(&user))
	bite.CanBeSilent(cmd)

	return cmd
}

//NewDeleteUserCommand deletes a new user
func NewDeleteUserCommand() *cobra.Command {
	var username string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a user",
		Example:          "users delete --username john",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"username": username}); err != nil {
				return err
			}

			if err := config.Client.DeleteUser(username); err != nil {
				return fmt.Errorf("Failed to delete user [%s]. [%s]", username, err.Error())
			}
			return bite.PrintInfo(cmd, "User [%s] deleted.", username)
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "User username")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

//NewPasswordUserCommand updates user password
func NewPasswordUserCommand() *cobra.Command {
	var username, password string

	cmd := &cobra.Command{
		Use:              "password",
		Short:            "Updates the password of a user",
		Example:          "users password --username john --secret secretpass",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{
				"username": username,
				"password": password,
			}); err != nil {
				return err
			}

			if err := config.Client.UpdateUserPassword(username, password); err != nil {
				return fmt.Errorf("Failed to update user's password [%s]. [%s]", username, err.Error())
			}
			return bite.PrintInfo(cmd, "User password [%s] updated.", username)
		},
	}

	cmd.Flags().StringVar(&username, "username", "", "User username")
	cmd.Flags().StringVar(&password, "secret", "", "User password")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}
