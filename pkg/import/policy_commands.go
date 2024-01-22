package imports

import (
	"fmt"

	"github.com/kataras/golog"
	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	"github.com/spf13/cobra"
)

// NewImportPoliciesCommand creates `import policies` ommand
func NewImportPoliciesCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "policies",
		Short:            "policies",
		Example:          `import policies --dir /my-landscape`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			path = fmt.Sprintf("%s/%s", path, pkg.PoliciesPath)
			if err := loadPolicies(config.Client, cmd, path); err != nil {
				golog.Errorf("Failed to load policies. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func loadPolicies(client *api.Client, cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading data policies from [%s]", loadpath)
	files, err := utils.FindFiles(loadpath)
	if err != nil {
		return err
	}

	polices, err := client.GetPolicies()
	if err != nil {
		return err
	}

	for _, file := range files {

		var policy api.DataPolicyRequest
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &policy); err != nil {
			return err
		}

		found := false

		for _, p := range polices {
			if p.Name == policy.Name {
				found = true

				payload := api.DataPolicyUpdateRequest{
					ID:          p.ID,
					Name:        p.Name,
					Category:    p.Category,
					ImpactType:  p.ImpactType,
					Obfuscation: p.Obfuscation,
					Fields:      p.Fields,
				}

				if err := client.UpdatePolicy(payload); err != nil {
					golog.Errorf("Error updating data policy [%s]. [%s]", p.Name, err.Error())
					return err
				}
				golog.Infof("Updated policy [%s]", p.Name)
			}
		}

		if !found {
			if err := client.CreatePolicy(policy); err != nil {
				golog.Errorf("Error creating data policy [%s]. [%s]", policy.Name, err.Error())
				return err
			}
			golog.Infof("Created data policy [%s]", policy.Name)
		}
	}

	return nil
}
