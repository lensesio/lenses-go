package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportAclsCommand creates `export acls` command
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

	if len(acls) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "no available ACLs for export\n")
		return nil
	}

	if err := utils.WriteFile(landscapeDir, pkg.AclsPath, fileName, output, acls); err != nil {
		return err
	}

	exportPath := fmt.Sprintf("%s/%s/%s", landscapeDir, pkg.AclsPath, fileName)
	fmt.Fprintf(cmd.OutOrStdout(), "ACLs have been successfully exported at %s\n", exportPath)
	return nil
}
