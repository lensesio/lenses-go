package management

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

//NewServiceAccountsCommand creates the `groups` command
func NewServiceAccountsCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "serviceaccounts",
		Short:            "Manage service accounts",
		Example:          "serviceaccounts",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			svcaccs, err := config.Client.GetServiceAccounts()
			if err != nil {
				golog.Errorf("Failed to find groups. [%s]", err.Error())
				return err
			}
			return bite.PrintObject(cmd, svcaccs)
		},
	}

	root.AddCommand(NewGetServiceAccountCommand())
	root.AddCommand(NewCreateServiceAccountCommand())
	root.AddCommand(NewUpdateServiceAccountCommand())
	root.AddCommand(NewDeleteServiceAccountCommand())
	root.AddCommand(NewRevokeServiceAccountCommand())
	return root
}

//NewGetServiceAccountCommand creates `serviceaccounts get`
func NewGetServiceAccountCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get the service account by provided name",
		Example: `
serviceaccounts get --name=svcacc
serviceaccounts get --name=svcacc
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}
			svcacc, err := config.Client.GetServiceAccount(name)
			if err != nil {
				return fmt.Errorf("Failed to find service account. [%s]", err.Error())
			}
			return bite.PrintObject(cmd, svcacc)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", `Service account name`)
	bite.CanPrintJSON(cmd)
	return cmd
}

//NewCreateServiceAccountCommand creates`serviceaccounts create`
func NewCreateServiceAccountCommand() *cobra.Command {
	var svcacc api.ServiceAccount

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a service account",
		Example: `
serviceaccounts create --name john --owner admin --groups MyGroup1 --groups MyGroup2
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateCreateUpdateSvcAcc(cmd, &svcacc); err != nil {
				return err
			}
			payload, err := config.Client.CreateServiceAccount(&svcacc)
			if err != nil {
				return fmt.Errorf("Failed to create service account [%s]. [%s]", svcacc.Name, err.Error())
			}

			return bite.PrintInfo(cmd, "Service Account [%s] created with token [%s]", svcacc.Name, payload.Token)
		},
	}
	addCreateUpdateSvcAccFlags(cmd, &svcacc)

	return cmd
}

//NewUpdateServiceAccountCommand creates`serviceaccounts update`
func NewUpdateServiceAccountCommand() *cobra.Command {
	var svcacc api.ServiceAccount

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a service account",
		Example: `
serviceaccounts update --name john --owner admin --groups MyGroup1 --groups MyGroup2
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateCreateUpdateSvcAcc(cmd, &svcacc); err != nil {
				return err
			}
			if err := config.Client.UpdateServiceAccount(&svcacc); err != nil {
				return fmt.Errorf("Failed to create service account [%s]. [%s]", svcacc.Name, err.Error())
			}

			return bite.PrintInfo(cmd, "Service account [%s] updated", svcacc.Name)
		},
	}
	addCreateUpdateSvcAccFlags(cmd, &svcacc)

	return cmd
}

//NewDeleteServiceAccountCommand creates  `serviceaccounts delete`
func NewDeleteServiceAccountCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a service account",
		Example:          "serviceaccounts delete --name svcaccount",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}

			if err := config.Client.DeleteServiceAccount(name); err != nil {
				return fmt.Errorf("Failed to delete service account [%s]. [%s]", name, err.Error())
			}
			return bite.PrintInfo(cmd, "Service account [%s] deleted.", name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Service account name")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

//NewRevokeServiceAccountCommand creates  `serviceaccounts revoke`
func NewRevokeServiceAccountCommand() *cobra.Command {
	var name, token string

	cmd := &cobra.Command{
		Use:              "revoke",
		Short:            "Revoke a service account token",
		Example:          "serviceaccounts revoke--name svcaccount",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}
			payload, err := config.Client.RevokeServiceAccountToken(name, token)
			if err != nil {
				return fmt.Errorf("Failed to revoke service account token [%s]. [%s]", name, err.Error())
			}
			return bite.PrintInfo(cmd, "Service account token [%s] revoked. New token [%s]", name, payload.Token)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Service account name")
	cmd.Flags().StringVar(&token, "token", "", "Your own manual service account token. Otherwise will be autogenerated")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func validateCreateUpdateSvcAcc(cmd *cobra.Command, svcacc *api.ServiceAccount) error {
	flags := bite.FlagPair{
		"name":   svcacc.Name,
		"groups": svcacc.Groups,
	}
	if err := bite.CheckRequiredFlags(cmd, flags); err != nil {
		return err
	}
	return nil
}

func addCreateUpdateSvcAccFlags(cmd *cobra.Command, svcacc *api.ServiceAccount) {
	cmd.Flags().StringVar(&svcacc.Name, "name", "", "Service account name")
	cmd.Flags().StringVar(&svcacc.Owner, "owner", "", "Service account owner")
	cmd.Flags().StringArrayVar(&svcacc.Groups, "groups", []string{}, "Service account groups")

	bite.Prepend(cmd, bite.FileBind(&svcacc))
	bite.CanBeSilent(cmd)
}
