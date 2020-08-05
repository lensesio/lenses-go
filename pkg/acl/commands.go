package acl

import (
	"sort"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var acl api.ACL
var acls []api.ACL

// NewGetACLsCommand creates the `acls` command
func NewGetACLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "Print the list of the available Kafka Access Control Lists",
		Example:          "acls",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// if the API changes: bite.FriendlyError(cmd, errResourceNotAccessibleMessage, "no authorizer is configured on the broker")
			acls, err := config.Client.GetACLs()
			if err != nil {
				golog.Errorf("Failed to retrieve acls. [%s]", err.Error())
				return err
			}

			sort.Slice(acls, func(i, j int) bool {
				//	return acls[i].Operation < acls[j].Operation
				return acls[i].ResourceName < acls[j].ResourceName
			})

			return bite.PrintObject(cmd, acls)
		},
	}

	bite.CanPrintJSON(cmd)

	return cmd
}

// NewACLGroupCommand creates the `acl` command
func NewACLGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "acl",
		Short:            "Manage Access Control List",
		Example:          "acl -h",
		TraverseChildren: true,
	}

	var (
		childrenRequiredFlags = func() bite.FlagPair {
			return bite.FlagPair{"resource-type": acl.ResourceType, "resource-name": acl.ResourceName, "principal": acl.Principal, "operation": acl.Operation}
		}
	)

	childrenFlagSet := pflag.NewFlagSet("acl", pflag.ExitOnError)
	childrenFlagSet.Var(bite.NewFlagVar(&acl.ResourceType), "resource-type", "The resource type: Topic, Cluster, Group or TRANSACTIONALID")
	childrenFlagSet.Var(bite.NewFlagVar(&acl.ResourceType), "resourceType", "The resource type: Topic, Cluster, Group or TRANSACTIONALID --- Deprecated ---")
	childrenFlagSet.StringVar(&acl.ResourceName, "resource-name", "", "The name of the resource")
	childrenFlagSet.StringVar(&acl.ResourceName, "resourceName", "", "The name of the resource --- Deprecated ---")
	childrenFlagSet.StringVar(&acl.Principal, "principal", "", "The name of the principal")
	childrenFlagSet.Var(bite.NewFlagVar(&acl.PermissionType), "permission-type", "Allow or Deny")
	childrenFlagSet.Var(bite.NewFlagVar(&acl.PermissionType), "permissionType", "Allow or Deny --- Deprecated ---")
	childrenFlagSet.StringVar(&acl.Host, "acl-host", "", "The acl host, can be empty to apply to all")
	childrenFlagSet.StringVar(&acl.Host, "aclHost", "", "The acl host, can be empty to apply to all --- Deprecated ---")
	childrenFlagSet.Var(bite.NewFlagVar(&acl.Operation), "operation", "The allowed operation: All, Read, Write, Describe, Create, Delete, DescribeConfigs, AlterConfigs, ClusterAction, IdempotentWrite or Alter")

	createCommand := NewCreateOrUpdateACLCommand(config.Client, childrenFlagSet, childrenRequiredFlags)
	bite.CanBeSilent(createCommand)
	bite.Prepend(createCommand, bite.FileBind(&acls))

	root.AddCommand(createCommand)
	root.AddCommand(NewDeleteACLCommand(config.Client, childrenFlagSet, childrenRequiredFlags))

	return root
}

//NewCreateOrUpdateACLCommand creates `acl set` command
func NewCreateOrUpdateACLCommand(client *api.Client, childrenFlagSet *pflag.FlagSet, requiredFlags func() bite.FlagPair) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"}, // acl create or acl update or acl set.
		Short:            "Sets, create or update Access Control Lists",
		Example:          `acl set --resource-type="Topic" --resource-name="transactions" --principal="principalType:principalName" --permission-type="Allow" --acl-host="*" --operation="Read"`,
		TraverseChildren: true,
		RunE: bite.Join(
			func(cmd *cobra.Command, args []string) error {
				if len(acls) > 0 {
					for _, acl := range acls {
						if err := client.CreateOrUpdateACL(acl); err != nil {
							golog.Errorf("Failed to create acl. [%s]", err.Error())
							return err
						}
					}
					return bite.PrintInfo(cmd, "ACLs created")
				}
				if err := bite.CheckRequiredFlags(cmd, requiredFlags()); err != nil {
					return err
				}
				if err := client.CreateOrUpdateACL(acl); err != nil {
					return err
				}
				return bite.PrintInfo(cmd, "ACL created")
			}),
	}
	cmd.Flags().AddFlagSet(childrenFlagSet)
	return cmd
}

//NewDeleteACLCommand creates `acl delete` command
func NewDeleteACLCommand(client *api.Client, childrenFlagSet *pflag.FlagSet, requiredFlags func() bite.FlagPair) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an Access Control List",
		Example:          `acl delete ./acl_to_be_deleted.json or .yml or acl delete --resource-type="Topic" --resource-name="transactions" --principal="principalType:principalName" --permission-type="Allow" --acl-host="*" --operation="Read"`,
		TraverseChildren: true,
		RunE: bite.Join(
			bite.FileBind(&acl),
			bite.RequireFlags(requiredFlags),
			func(cmd *cobra.Command, args []string) error {
				if err := client.DeleteACL(acl); err != nil {
					golog.Errorf("Failed to delete acl. [%s]", err.Error())
					return err
				}

				return bite.PrintInfo(cmd, "ACL deleted if it exists")
			}),
	}

	cmd.Flags().AddFlagSet(childrenFlagSet)
	bite.CanBeSilent(cmd)

	return cmd
}
