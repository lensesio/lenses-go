package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

//NewExportGroupsCommand creates `export users`
func NewExportGroupsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "groups",
		Short:            "export groups",
		Example:          `export groups`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			groups, err := config.Client.GetGroups()
			if err != nil {
				golog.Errorf("Failed to find groups. [%s]", err.Error())
				return err
			}
			if err := writeGroups(cmd, &groups); err != nil {
				golog.Errorf("Error writing Users. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeGroups(cmd *cobra.Command, groups *[]api.Group) error {

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("groups.%s", strings.ToLower(output))
	return utils.WriteFile(landscapeDir, pkg.GroupsPath, fileName, output, groups)
}
