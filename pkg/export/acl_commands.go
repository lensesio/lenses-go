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

//NewExportAclsCommand creates `export acls` command
func NewExportAclsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "export acls",
		Example:          `export acls`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			if err := writeACLs(cmd, config.Client); err != nil {
				golog.Errorf("Error writing ACLS. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract dependencies, topics, acls, quotas, alerts")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func writeACLs(cmd *cobra.Command, client *api.Client) error {

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("acls.%s", strings.ToLower(output))

	acls, err := client.GetACLs()

	if err != nil {
		return err
	}

	return utils.WriteFile(landscapeDir, pkg.AclsPath, fileName, output, acls)
}
