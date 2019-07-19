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

//NewExportUsersCommand creates `export users`
func NewExportUsersCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "users",
		Short:            "export users",
		Example:          `export users`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			users, err := config.Client.GetUsers()
			if err != nil {
				golog.Errorf("Failed to find users. [%s]", err.Error())
				return err
			}
			if err := writeUsers(cmd, &users); err != nil {
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

func writeUsers(cmd *cobra.Command, users *[]api.UserMember) error {

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("users.%s", strings.ToLower(output))
	return utils.WriteFile(landscapeDir, pkg.UsersPath, fileName, output, users)
}
