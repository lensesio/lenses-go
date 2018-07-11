package main

import (
	"github.com/landoop/lenses-go"
	"sort"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func init() {
	app.AddCommand(newGetACLsCommand())
	app.AddCommand(newACLGroupCommand())
}

func newGetACLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "Print the list of the available Apache Kafka Access Control Lists",
		Example:          "acls",
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, _ []string) error {
			acls, err := client.GetACLs()
			if err != nil {
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
		Short:            "Work with Apache Kafka Access Control List",
		Example:          "acl -h",
		TraverseChildren: true,
	}

	var (
		acl                   lenses.ACL
		childrenRequiredFlags = func() bite.FlagPair {
			return bite.FlagPair{"resourceType": acl.ResourceType, "resourceName": acl.ResourceName, "principal": acl.Principal, "operation": acl.Operation}
		}
	)

	childrenFlagSet := pflag.NewFlagSet("acl", pflag.ExitOnError)
	childrenFlagSet.Var(bite.NewFlagVar(&acl.ResourceType), "resourceType", "the resource type: Topic, Cluster, Group or TRANSACTIONALID")
	childrenFlagSet.StringVar(&acl.ResourceName, "resourceName", "", "the name of the resource")
	childrenFlagSet.StringVar(&acl.Principal, "principal", "", "the name of the principal")
	childrenFlagSet.Var(bite.NewFlagVar(&acl.PermissionType), "permissionType", "Allow or Deny")
	childrenFlagSet.StringVar(&acl.Host, "acl-host", "", "the acl host, can be empty to apply to all")
	childrenFlagSet.Var(bite.NewFlagVar(&acl.Operation), "operation", "the allowed operation: All, Read, Write, Describe, Create, Delete, DescribeConfigs, AlterConfigs, ClusterAction, IdempotentWrite or Alter")

	root.AddCommand(newCreateOrUpdateACLCommand(&acl, childrenFlagSet, childrenRequiredFlags))
	root.AddCommand(newDeleteACLCommand(&acl, childrenFlagSet, childrenRequiredFlags))
	return root
}

func newCreateOrUpdateACLCommand(acl *lenses.ACL, childrenFlagSet *pflag.FlagSet, requiredFlags func() bite.FlagPair) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"}, // acl create or acl update or acl set.
		Short:            "Sets, create or update, an Apache Kafka Access Control List",
		Example:          `acl set --resourceType="Topic" --resourceName="transactions" --principal="principalType:principalName" --permissionType="Allow" --acl-host="*" --operation="Read"`,
		TraverseChildren: true,
		RunE: bite.Join(
			bite.FileBind(acl),
			bite.RequireFlags(requiredFlags),
			func(cmd *cobra.Command, args []string) error {
				if err := client.CreateOrUpdateACL(*acl); err != nil {
					return err
				}

				return bite.PrintInfo(cmd, "ACL created")
			}),
	}

	cmd.Flags().AddFlagSet(childrenFlagSet)

	return cmd
}

func newDeleteACLCommand(acl *lenses.ACL, childrenFlagSet *pflag.FlagSet, requiredFlags func() bite.FlagPair) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an Apache Kafka Access Control List",
		Example:          `acl delete ./acl_to_be_deleted.json or .yml or acl delete --resourceType="Topic" --resourceName="transactions" --principal="principalType:principalName" --permissionType="Allow" --acl-host="*" --operation="Read"`,
		TraverseChildren: true,
		RunE: bite.Join(
			bite.FileBind(acl),
			bite.RequireFlags(requiredFlags),
			func(cmd *cobra.Command, args []string) error {
				if err := client.DeleteACL(*acl); err != nil {
					bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to delete, acl does not exist")
					return err
				}

				return bite.PrintInfo(cmd, "ACL deleted")
			}),
	}

	cmd.Flags().AddFlagSet(childrenFlagSet)
	return cmd
}
