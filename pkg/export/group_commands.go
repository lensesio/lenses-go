package export

import (
	"fmt"
	"hash/fnv"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportGroupsCommand creates `export users`
func NewExportGroupsCommand() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:              "groups",
		Short:            "export groups",
		Example:          `export groups`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			if err := writeGroups(cmd, name); err != nil {
				golog.Errorf("Error writing Users. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().StringVar(&name, "name", "", "The group name to extract")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeGroups(cmd *cobra.Command, groupName string) error {

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	if groupName != "" {
		group, err := config.Client.GetGroup(groupName)
		if err != nil {
			return err
		}

		h := fnv.New64a()
		_, err = h.Write([]byte(group.Name))
		if err != nil {
			return err
		}
		fileName := fmt.Sprintf("groups-%s-%08x.%s", strings.ToLower(group.Name), uint32(h.Sum64()), strings.ToLower(output))
		return utils.WriteFile(landscapeDir, pkg.GroupsPath, fileName, output, group)
	}
	groups, err := config.Client.GetGroups()
	if err != nil {
		return err
	}

	h := fnv.New64a()

	for _, group := range groups {

		_, err = h.Write([]byte(group.Name))
		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("groups-%s-%08x.%s", strings.ToLower(group.Name), uint32(h.Sum64()), strings.ToLower(output))

		if groupName != "" && group.Name == groupName {
			return utils.WriteFile(landscapeDir, pkg.GroupsPath, fileName, output, group)
		}

		err := utils.WriteFile(landscapeDir, pkg.GroupsPath, fileName, output, group)
		if err != nil {
			return err
		}
	}

	return nil
}
