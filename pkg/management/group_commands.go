package management

import (
	"encoding/json"
	"fmt"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

//NewGroupsCommand creates the `groups` command
func NewGroupsCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "groups",
		Short:            "Manage groups",
		Example:          "groups",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			groups, err := config.Client.GetGroups()
			if err != nil {
				golog.Errorf("Failed to find groups. [%s]", err.Error())
				return err
			}
			return bite.PrintObject(cmd, groups)
		},
	}

	root.AddCommand(NewGetGroupCommand())
	root.AddCommand(NewCreateGroupCommand())
	root.AddCommand(NewDeleteGroupCommand())
	root.AddCommand(NewUpdateGroupCommand())
	root.AddCommand(NewCloneGroupCommand())
	return root
}

//NewGetGroupCommand creates `groups get`
func NewGetGroupCommand() *cobra.Command {
	var (
		groupName     string
		namespaceOnly bool
	)

	cmd := &cobra.Command{
		Use:   "get",
		Short: "Get the group by provided name",
		Example: `
groups get --name=MyGroup
groups get --name=MyGroup --dataNamespaces
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": groupName}); err != nil {
				return err
			}
			group, err := config.Client.GetGroup(groupName)
			if err != nil {
				golog.Errorf("Failed to find group. [%s]", err.Error())
				return err
			}
			if namespaceOnly {
				return bite.PrintObject(cmd, group.Namespaces)
			}
			return bite.PrintObject(cmd, group)
		},
	}

	cmd.Flags().StringVar(&groupName, "name", "", `Group name`)
	cmd.Flags().BoolVar(&namespaceOnly, "dataNamespaces", false, `Print data namespaces only`)
	bite.CanPrintJSON(cmd)
	return cmd
}

//NewCreateGroupCommand creates a new group
func NewCreateGroupCommand() *cobra.Command {
	var (
		group         api.Group
		namespacesRaw string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a group",
		Example: `
groups create --name MyGroup --description "My test group" --applicationPermissions ViewKafkaConsumers --applicationPermissions ManageKafkaConsumers --applicationPermissions ViewConnectors --connectClustersPermissions dev,prod --dataNamespaces '[{"wildcards":["*"],"permissions":["CreateTopic","DropTopic","ConfigureTopic","QueryTopic","ShowTopic","ViewSchema","InsertData","DeleteData","UpdateSchema"],"system":"Kafka","instance":"Dev"}]'
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateCreateUpdate(cmd, &namespacesRaw, &group); err != nil {
				return err
			}

			if err := config.Client.CreateGroup(&group); err != nil {
				return fmt.Errorf("Failed to create group [%s]. [%s]", group.Name, err.Error())
			}
			return bite.PrintInfo(cmd, "Group [%s] created", group.Name)
		},
	}

	addCreateUpdateFlags(cmd, &namespacesRaw, &group)

	return cmd
}

//NewUpdateGroupCommand creates a new group
func NewUpdateGroupCommand() *cobra.Command {
	var (
		group         api.Group
		namespacesRaw string
	)

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Update a group",
		Example: `
groups update --name MyGroup --description "My test group" --applicationPermissions ViewKafkaConsumers --applicationPermissions ManageKafkaConsumers --applicationPermissions ViewConnectors --connectClustersPermissions dev,prod --dataNamespaces'[{"wildcards":["*"],"permissions":["CreateTopic","DropTopic","ConfigureTopic","QueryTopic","ShowTopic","ViewSchema","InsertData","DeleteData","UpdateSchema"],"system":"Kafka","instance":"Dev"}]'
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateCreateUpdate(cmd, &namespacesRaw, &group); err != nil {
				return err
			}

			if err := config.Client.UpdateGroup(&group); err != nil {
				return fmt.Errorf("Failed to update group [%s]. [%s]", group.Name, err.Error())
			}

			return bite.PrintInfo(cmd, "Group [%s] updated", group.Name)
		},
	}

	addCreateUpdateFlags(cmd, &namespacesRaw, &group)

	return cmd
}

//NewDeleteGroupCommand creates a new group
func NewDeleteGroupCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a group",
		Example:          "groups delete --name MyGroup",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"name": name}); err != nil {
				return err
			}

			if err := config.Client.DeleteGroup(name); err != nil {
				return fmt.Errorf("Failed to delete group [%s]. [%s]", name, err.Error())
			}
			return bite.PrintInfo(cmd, "Group [%s] deleted.", name)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Group name")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

//NewCloneGroupCommand clones a group
func NewCloneGroupCommand() *cobra.Command {
	var name, cloneName string

	cmd := &cobra.Command{
		Use:              "clone",
		Short:            "Clone a group",
		Example:          "groups clone --name MyGroup --cloneName MyClonedGroup",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{
				"name":      name,
				"cloneName": cloneName,
			}); err != nil {
				return err
			}

			if err := config.Client.CloneGroup(name, cloneName); err != nil {
				return fmt.Errorf("Failed to clone group [%s]. [%s]", name, err.Error())
			}
			return bite.PrintInfo(cmd, "Group [%s] cloned to [%s].", name, cloneName)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Group name")
	cmd.Flags().StringVar(&cloneName, "cloneName", "", "Name for the cloned group")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func validateCreateUpdate(cmd *cobra.Command, namespacesRaw *string, group *api.Group) error {

	flags := bite.FlagPair{
		"name": group.Name,
	}
	if err := bite.CheckRequiredFlags(cmd, flags); err != nil {
		return err
	}

	if len(group.ScopedPermissions) == 0 {
		return fmt.Errorf("required flag \"applicationPermissions\" not set")
	}
	if *namespacesRaw != "" {
		if err := bite.TryReadFile(*namespacesRaw, &group.Namespaces); err != nil {
			// from flag as json.
			if err = json.Unmarshal([]byte(*namespacesRaw), &group.Namespaces); err != nil {
				return fmt.Errorf("Unable to unmarshal the data namespaces: [%v]", err)
			}
		}
	}
	return nil
}

func addCreateUpdateFlags(cmd *cobra.Command, namespacesRaw *string, group *api.Group) {
	cmd.Flags().StringVar(&group.Name, "name", "", "Group name")
	cmd.Flags().StringVar(&group.Description, "description", "", "Group description")
	cmd.Flags().StringArrayVar(&group.ScopedPermissions, "applicationPermissions", []string{}, "Group application permissions")
	cmd.Flags().StringArrayVar(&group.AdminPermissions, "adminPermissions", []string{}, "Group admin permissions")
	cmd.Flags().StringVar(namespacesRaw, "dataNamespaces", "", `Group data namespaces: "[{"wildcards":["*"],"permissions":["CreateTopic","DropTopic","ConfigureTopic","QueryTopic","ShowTopic","ViewSchema","InsertData","DeleteData","UpdateSchema"],"system":"Kafka","instance":"Dev"}]"`)
	cmd.Flags().StringSliceVar(&group.ConnectClustersPermissions, "connectClustersPermissions", nil, "Connect clusters access")

	bite.Prepend(cmd, bite.FileBind(&group))
	bite.CanBeSilent(cmd)
}
