package main

import (
	"sort"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var acl lenses.ACL
var acls []lenses.ACL

func init() {
	app.AddCommand(newGetACLsCommand())
	app.AddCommand(newACLGroupCommand())
}

func newGetACLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "Print the list of the available Kafka Access Control Lists",
		Example:          "acls",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			// if the API changes: bite.FriendlyError(cmd, errResourceNotAccessibleMessage, "no authorizer is configured on the broker")
			acls, err := client.GetACLs()
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

func newACLGroupCommand() *cobra.Command {
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
	childrenFlagSet.StringVar(&acl.ResourceName, "resource-name", "", "The name of the resource")
	childrenFlagSet.StringVar(&acl.Principal, "principal", "", "The name of the principal")
	childrenFlagSet.Var(bite.NewFlagVar(&acl.PermissionType), "permission-type", "Allow or Deny")
	childrenFlagSet.StringVar(&acl.Host, "acl-host", "", "The acl host, can be empty to apply to all")
	childrenFlagSet.Var(bite.NewFlagVar(&acl.Operation), "operation", "The allowed operation: All, Read, Write, Describe, Create, Delete, DescribeConfigs, AlterConfigs, ClusterAction, IdempotentWrite or Alter")

	root.AddCommand(newCreateOrUpdateACLCommand(childrenFlagSet, childrenRequiredFlags))
	root.AddCommand(newDeleteACLCommand(childrenFlagSet, childrenRequiredFlags))
	return root
}

func newCreateOrUpdateACLCommand(childrenFlagSet *pflag.FlagSet, requiredFlags func() bite.FlagPair) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"}, // acl create or acl update or acl set.
		Short:            "Sets, create or update Access Control Lists",
		Example:          `acl set --resource-type="Topic" --resource-name="transactions" --principal="principalType:principalName" --permission-type="Allow" --acl-host="*" --operation="Read"`,
		TraverseChildren: true,
		RunE: bite.Join(
			bite.FileBind(&acls),
			bite.RequireFlags(requiredFlags),
			func(cmd *cobra.Command, args []string) error {

				if len(acls) > 0 {
					for _, acl := range acls {
						if err := client.CreateOrUpdateACL(acl); err != nil {
							golog.Errorf("Failed to create acl. [%s]", err.Error())
							return err
						}
						bite.PrintInfo(cmd, "ACL created")
					}
					return bite.PrintInfo(cmd, "ACL created")
				}

				return bite.PrintInfo(cmd, "ACL created")

			}),
	}

	cmd.Flags().AddFlagSet(childrenFlagSet)

	bite.CanBeSilent(cmd)
	return cmd
}

func newDeleteACLCommand(childrenFlagSet *pflag.FlagSet, requiredFlags func() bite.FlagPair) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an Access Control List",
		Example:          `acl delete ./acl_to_be_deleted.json or .yml or acl delete --resourceType="Topic" --resourceName="transactions" --principal="principalType:principalName" --permission-type="Allow" --acl-host="*" --operation="Read"`,
		TraverseChildren: true,
		RunE: bite.Join(
			bite.FileBind(&acl),
			bite.RequireFlags(requiredFlags),
			func(cmd *cobra.Command, args []string) error {
				if err := client.DeleteACL(acl); err != nil {
					golog.Errorf("Failed to delete acl. [%s]", err.Error())
					return err
				}

				return bite.PrintInfo(cmd, "ACL deleted")
			}),
	}

	cmd.Flags().AddFlagSet(childrenFlagSet)
	bite.CanBeSilent(cmd)

	return cmd
}
