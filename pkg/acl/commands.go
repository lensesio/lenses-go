package acl

import (
	"errors"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"
)

var aclFlags *pflag.FlagSet

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
	cmd := &cobra.Command{
		Use:              "acl",
		Short:            "Manage Access Control List",
		Example:          "acl -h",
		TraverseChildren: true,
	}

	// By creating a group of flags (flag set) we can then query based on that filtering out
	// other flags coming from parent commands.
	aclFlags = pflag.NewFlagSet("acl", pflag.ExitOnError)

	aclFlags.String("permission-type", "", "Kafka ACL permission type e.g. 'allow', 'deny', etc. (REQUIRED)")
	aclFlags.String("principal", "", "The name of the principal (REQUIRED)")
	aclFlags.String("operation", "", "Kafka ACL operation, e.g. 'all', 'read', 'write', etc. (REQUIRED)")
	aclFlags.String("resource-type", "", "Kafka ACL resource type e.g. 'topic', 'cluster', etc. (REQUIRED)")
	aclFlags.String("pattern-type", "", "Kafka ACL pattern, e.g. 'literal', 'prefixed', etc. (REQUIRED)")
	aclFlags.String("resource-name", "", "The name of the resource (REQUIRED)")
	aclFlags.String("acl-host", "*", "The ACL host")

	// Add the ACL flag set to root 'acl' command so it can be shared with sub-commands
	cmd.PersistentFlags().AddFlagSet(aclFlags)

	// Link "set" and "delete" sub-commands to root 'acl' command
	cmd.AddCommand(NewCreateOrUpdateACLCommand())
	cmd.AddCommand(NewDeleteACLCommand())

	return cmd
}

//NewCreateOrUpdateACLCommand creates `acl set` command
func NewCreateOrUpdateACLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"}, // acl create or acl update or acl set.
		Short:            "Set/create or update Access Control Lists",
		Example:          `acl set --resource-type="Topic" --resource-name="transactions" --principal="principalType:principalName" --permission-type="Allow" --acl-host="*" --operation="Read" --pattern-type="literal"`,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var acl api.ACL
			acl, err := populateACL(cmd, args)
			if err != nil {
				return err
			}

			if err := config.Client.CreateOrUpdateACL(acl); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ACL \"%s\" was created/updated successfuly\n", acl)

			return nil
		},
	}

	return cmd
}

//NewDeleteACLCommand creates `acl delete` command
func NewDeleteACLCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "delete",
		Aliases:          []string{"rm", "del"},
		Short:            "Delete an Access Control List",
		Example:          `acl delete ./acl_to_be_deleted.json or .yml or acl delete --resource-type="Topic" --resource-name="transactions" --principal="principalType:principalName" --permission-type="Allow" --acl-host="*" --operation="Read"`,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			var acl api.ACL
			acl, err := populateACL(cmd, args)
			if err != nil {
				return err
			}

			if err := config.Client.DeleteACL(acl); err != nil {
				return fmt.Errorf("failed to delete ACL '%s'. [%s]", acl, err.Error())
			}

			fmt.Fprintf(cmd.OutOrStdout(), "ACL \"%s\" was deleted successfully\n", acl)
			return nil
		},
	}
	return cmd
}

// populateACL function will try populate an ACL struct either from a yaml file or from cli flags
func populateACL(cmd *cobra.Command, args []string) (api.ACL, error) {
	var acl api.ACL
	switch len(args) {
	// Handle case when user opts to use CLI flags (flags do not count as arguments)
	case 0:
		acl.Host, _ = cmd.Flags().GetString("acl-host")
		operation, _ := cmd.Flags().GetString("operation")
		acl.Operation = api.ACLOperation(operation)
		acl.PatternType, _ = cmd.Flags().GetString("pattern-type")
		permissionType, _ := cmd.Flags().GetString("permission-type")
		acl.PermissionType = api.ACLPermissionType(permissionType)
		acl.Principal, _ = cmd.Flags().GetString("principal")
		acl.ResourceName, _ = cmd.Flags().GetString("resource-name")
		resourceType, _ := cmd.Flags().GetString("resource-type")
		acl.ResourceType = api.ACLResourceType(resourceType)

		if err := isACLPopulated(acl); err != nil {
			return acl, err
		}

		return acl, nil
	// Handle case when user opts to use a file (only one file argument is allowed)
	case 1:
		// Check that that one argument passed is a reference to a YML file and try to unmarshall it
		matched, _ := regexp.MatchString("(.yaml|.yml)$", args[0])
		if !matched {
			return api.ACL{}, errors.New("expecting a file argument ending in .yml or .yaml")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "loading values from YAML file '%s'...\n", args[0])
		yamlFile, err := ioutil.ReadFile(args[0])
		if err != nil {
			return api.ACL{}, err
		}
		if err := yaml.Unmarshal(yamlFile, &acl); err != nil {
			return api.ACL{}, err
		}
		if err = isACLPopulated(acl); err != nil {
			return acl, err
		}
		return acl, nil
	default:
		return api.ACL{}, errors.New("multiple cli arguments found, please check proper command usage with 'acl -h'")
	}
}

// isACLPopulated checks whether all required fields are populated
func isACLPopulated(acl api.ACL) error {

	if acl.Operation == "" || acl.PatternType == "" || acl.PermissionType == "" || acl.Principal == "" || acl.ResourceName == "" || acl.ResourceType == "" {
		return errors.New("missing ACL flag(s), please run 'lenses-cli acl -h' for required flags")
	}
	return nil
}
