package export

import (
	"fmt"
	"strings"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/lenses-go/pkg/utils"
	"github.com/spf13/cobra"
)

// NewExportPoliciesCommand creates `export policies` command
func NewExportPoliciesCommand() *cobra.Command {
	var name, ID string

	cmd := &cobra.Command{
		Use:              "policies",
		Short:            "export policies",
		Example:          `export policies --resource-name my-policy`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			client := config.Client

			setExecutionMode(client)
			checkFileFlags(cmd)
			if err := writePolicies(cmd, client, name, ID); err != nil {
				golog.Errorf("Error writing policies. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.Flags().BoolVar(&dependents, "dependents", false, "Extract dependencies, topics, acls, quotas, alerts")
	cmd.Flags().StringVar(&name, "resource-name", "", "The resource name to export")
	cmd.Flags().StringVar(&ID, "id", "", "The policy id to extract")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func writePolicies(cmd *cobra.Command, client *api.Client, name string, ID string) error {
	golog.Infof("Writing policies to [%s]", landscapeDir)
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	if ID != "" {
		policy, err := client.GetPolicy(ID)
		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("policies-%s.%s", strings.ToLower(policy.Name), strings.ToLower(output))
		request := client.PolicyAsRequest(policy)
		return utils.WriteFile(landscapeDir, pkg.PoliciesPath, fileName, output, request)
	}

	policies, err := client.GetPolicies()
	if err != nil {
		return err
	}

	for _, policy := range policies {
		fileName := fmt.Sprintf("policies-%s.%s", strings.ToLower(policy.Name), strings.ToLower(output))
		if name != "" && policy.Name == name {
			return utils.WriteFile(landscapeDir, pkg.PoliciesPath, fileName, output, policy)
		}

		err := utils.WriteFile(landscapeDir, pkg.PoliciesPath, fileName, output, policy)
		if err != nil {
			return err
		}
	}

	return nil
}
