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
		Example:          exampleString("acl"),
		TraverseChildren: true,
	}
	root.AddCommand(newCreateOrUpdateACLCommand())
	root.AddCommand(newDeleteACLCommand())

	return root
}

func newCreateOrUpdateACLCommand() *cobra.Command {
	var acl lenses.ACL

	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"}, // acl create or acl update or acl set.
		Short:            "Sets, create or update, an Apache Kafka Access Control List",
		Example:          exampleString(`acl set --resourceType="Topic" --resourceName="transactions" --principal="principalType:principalName" --permissionType="Allow" --host="*" --operation="Read"`),
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"resourceType": acl.ResourceType, "resourceName": acl.ResourceName, "principal": acl.Principal, "operation": acl.Operation}); err != nil {
				return err
			}

			if err := client.CreateOrUpdateACL(acl); err != nil {
				return err
			}

			return echo(cmd, "ACL created")
		},
	}

	cmd.Flags().Var(newVarFlag(&acl.ResourceType), "resourceType", "--resourceType The resource type, TOPIC, CLUSTER, GROUP, TRANSACTIONALID")
	cmd.Flags().StringVar(&acl.ResourceName, "resourceName", "", "--resourceName The name of the resource")
	cmd.Flags().StringVar(&acl.Principal, "principal", "", "--principal The name of the principal")
	cmd.Flags().Var(newVarFlag(&acl.PermissionType), "permissionType", "--permissionType ALLOW or deny")
	cmd.Flags().StringVar(&acl.Host, "host", "", "--host") // optional, defaults to "*".
	cmd.Flags().Var(newVarFlag(&acl.Operation), "operation", "--operation The allowed operation, ALL, READ, WRITE, DELETE, DESCRIBECONFIGS, ALTERCONFIGS, IDEMPOTENTWRITE")

	shouldTryLoadFile(cmd, &acl)
	canBeSilent(cmd)

	return cmd
}

func newDeleteACLCommand() *cobra.Command {
	var acl lenses.ACL

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an Apache Kafka Access Control List",
		Example:          exampleString(`acl delete ./acl_to_be_deleted.json or .yml or acl delete --resourceType="Topic" --resourceName="transactions" --principal="principalType:principalName" --permissionType="Allow" --host="*" --operation="Read"`),
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"resourceType": acl.ResourceType, "resourceName": acl.ResourceName, "principal": acl.Principal, "operation": acl.Operation}); err != nil {
				return err
			}

			if err := client.DeleteACL(acl); err != nil {
				errResourceNotFoundMessage = "unable to delete, acl does not exist"
				return err
			}

			return echo(cmd, "ACL deleted")
		},
	}

	cmd.Flags().Var(newVarFlag(&acl.ResourceType), "resourceType", "--resourceType The resource type, TOPIC, CLUSTER, GROUP, TRANSACTIONALID")
	cmd.Flags().StringVar(&acl.ResourceName, "resourceName", "", "--resourceName The name of the resource")
	cmd.Flags().StringVar(&acl.Principal, "principal", "", "--principal The name of the principal")
	cmd.Flags().Var(newVarFlag(&acl.PermissionType), "permissionType", "--permissionType ALLOW or deny")
	cmd.Flags().StringVar(&acl.Host, "host", "", "--host")
	cmd.Flags().Var(newVarFlag(&acl.Operation), "operation", "--operation The allowed operation, ALL, READ, WRITE, DELETE, DESCRIBECONFIGS, ALTERCONFIGS, IDEMPOTENTWRITE")

	shouldTryLoadFile(cmd, &acl)
	canBeSilent(cmd)

	return cmd
}
