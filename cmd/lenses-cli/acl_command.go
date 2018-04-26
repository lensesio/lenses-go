package main

import (
	"github.com/landoop/lenses-go"
	"github.com/spf13/pflag"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newGetACLsCommand())
	rootCmd.AddCommand(newACLGroupCommand())
}

func newGetACLsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "Print the list of the available Apache Kafka Access Control Lists",
		Example:          exampleString("acls"),
		TraverseChildren: true,
	}

	shouldPrintJSON(cmd, func() (interface{}, error) {
		return client.GetACLs()
	})

	return cmd
}

func newACLGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "acl",
		Short:            "Work with Apache Kafka Access Control List",
		Example:          exampleString("acl -h"),
		TraverseChildren: true,
	}

	var (
		acl                   lenses.ACL
		childrenRequiredFlags = func() flags {
			return flags{"resourceType": acl.ResourceType, "resourceName": acl.ResourceName, "principal": acl.Principal, "operation": acl.Operation}
		}
	)

	childrenFlagSet := newFlagSet("acl")
	childrenFlagSet.Var(newVarFlag(&acl.ResourceType), "resourceType", "the resource type: Topic, Cluster, Group or TRANSACTIONALID")
	childrenFlagSet.StringVar(&acl.ResourceName, "resourceName", "", "the name of the resource")
	childrenFlagSet.StringVar(&acl.Principal, "principal", "", "the name of the principal")
	childrenFlagSet.Var(newVarFlag(&acl.PermissionType), "permissionType", "Allow or Deny")
	childrenFlagSet.StringVar(&acl.Host, "acl-host", "", "the acl host, can be empty to apply to all")
	childrenFlagSet.Var(newVarFlag(&acl.Operation), "operation", "the allowed operation: All, Read, Write, Describe, Create, Delete, DescribeConfigs, AlterConfigs, ClusterAction, IdempotentWrite or Alter")

	root.AddCommand(newCreateOrUpdateACLCommand(&acl, childrenFlagSet, childrenRequiredFlags))
	root.AddCommand(newDeleteACLCommand(&acl, childrenFlagSet, childrenRequiredFlags))
	return root
}

func newCreateOrUpdateACLCommand(acl *lenses.ACL, childrenFlagSet *pflag.FlagSet, requiredFlags func() flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"}, // acl create or acl update or acl set.
		Short:            "Sets, create or update, an Apache Kafka Access Control List",
		Example:          exampleString(`acl set --resourceType="Topic" --resourceName="transactions" --principal="principalType:principalName" --permissionType="Allow" --acl-host="*" --operation="Read"`),
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.CreateOrUpdateACL(*acl); err != nil {
				return err
			}

			return echo(cmd, "ACL created")
		},
	}

	cmd.Flags().AddFlagSet(childrenFlagSet)

	canBeSilent(cmd)
	shouldTryLoadFile(cmd, acl)
	shouldCheckRequiredFlags(cmd, requiredFlags)

	return cmd
}

func newDeleteACLCommand(acl *lenses.ACL, childrenFlagSet *pflag.FlagSet, requiredFlags func() flags) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an Apache Kafka Access Control List",
		Example:          exampleString(`acl delete ./acl_to_be_deleted.json or .yml or acl delete --resourceType="Topic" --resourceName="transactions" --principal="principalType:principalName" --permissionType="Allow" --acl-host="*" --operation="Read"`),
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := client.DeleteACL(*acl); err != nil {
				errResourceNotFoundMessage = "unable to delete, acl does not exist"
				return err
			}

			return echo(cmd, "ACL deleted")
		},
	}

	cmd.Flags().AddFlagSet(childrenFlagSet)

	canBeSilent(cmd)
	shouldTryLoadFile(cmd, acl)
	shouldCheckRequiredFlags(cmd, requiredFlags)

	return cmd
}
