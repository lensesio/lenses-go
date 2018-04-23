package main

import (
	"github.com/landoop/lenses-go"

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

	shouldReturnJSON(cmd, func() (interface{}, error) {
		return client.GetACLs()
	})

	return cmd
}

func newACLGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "acl",
		Short:            "Work with an Apache Kafka Access Control Lists",
		Example:          exampleString("acl --help "),
		TraverseChildren: true,
	}

	var acl lenses.ACL

	root.AddCommand(newCreateOrUpdateACLCommand(&acl))
	root.AddCommand(newDeleteACLCommand(&acl))

	return visitChildren(root, func(cmd *cobra.Command) {
		cmd.Flags().Var(newVarFlag(&acl.ResourceType), "resourceType", "The resource type: Topic, Cluster, Group or TRANSACTIONALID")
		cmd.Flags().StringVar(&acl.ResourceName, "resourceName", "", "The name of the resource")
		cmd.Flags().StringVar(&acl.Principal, "principal", "", "The name of the principal")
		cmd.Flags().Var(newVarFlag(&acl.PermissionType), "permissionType", "Allow or Deny")
		cmd.Flags().StringVar(&acl.Host, "acl-host", "", "The acl host, can be empty to apply to all")
		cmd.Flags().Var(newVarFlag(&acl.Operation), "operation", "The allowed operation: All, Read, Write, Describe, Create, Delete, DescribeConfigs, AlterConfigs, ClusterAction, IdempotentWrite or Alter")

		shouldTryLoadFile(cmd, &acl)
	})
}

func newCreateOrUpdateACLCommand(acl *lenses.ACL) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"}, // acl create or acl update or acl set.
		Short:            "Sets, create or update, an Apache Kafka Access Control List",
		Example:          exampleString(`acl set --resourceType="Topic" --resourceName="transactions" --principal="principalType:principalName" --permissionType="Allow" --acl-host="*" --operation="Read"`),
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"resourceType": acl.ResourceType, "resourceName": acl.ResourceName, "principal": acl.Principal, "operation": acl.Operation}); err != nil {
				return err
			}

			if err := client.CreateOrUpdateACL(*acl); err != nil {
				return err
			}

			return echo(cmd, "ACL created")
		},
	}

	canBeSilent(cmd)

	return cmd
}

func newDeleteACLCommand(acl *lenses.ACL) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an Apache Kafka Access Control List",
		Example:          exampleString(`acl delete ./acl_to_be_deleted.json or .yml or acl delete --resourceType="Topic" --resourceName="transactions" --principal="principalType:principalName" --permissionType="Allow" --acl-host="*" --operation="Read"`),
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"resourceType": acl.ResourceType, "resourceName": acl.ResourceName, "principal": acl.Principal, "operation": acl.Operation}); err != nil {
				return err
			}

			if err := client.DeleteACL(*acl); err != nil {
				errResourceNotFoundMessage = "unable to delete, acl does not exist"
				return err
			}

			return echo(cmd, "ACL deleted")
		},
	}

	canBeSilent(cmd)

	return cmd
}
