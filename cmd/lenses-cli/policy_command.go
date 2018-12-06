package main

import (
	"strings"

	"github.com/landoop/lenses-go"
	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newGetPoliciesCommand())
	app.AddCommand(newPolicyGroupCommand())
}

func newGetPoliciesCommand() *cobra.Command {

	var name string

	cmd := &cobra.Command{
		Use:              "policies",
		Short:            "List all policies",
		Example:          `policies`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
 			
			result, err := client.GetPolicies()
			if err != nil {
				return err
			}

			for _, policy := range result {
				if name != "" && name == policy.Name {
					return bite.PrintObject(cmd, policy)
				}
			}

			if name != "" {
				golog.Errorf("Failed to retrieve policy [%s]. [%s]",name, err.Error())
				return err
			}

			return bite.PrintObject(cmd, result)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Policy name")
	cmd.AddCommand(newGetPoliciesImpactTypesCommand())
	cmd.AddCommand(newGetPoliciesObfuscationCommand())
	bite.CanPrintJSON(cmd)
	return cmd
}

func newPolicyGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "policy",
		Short:            "Manage a policy",
		Example:          `
policy create --name my-policy --category my-category --impact HIGH --redaction First-1 --fields myfield1,myfield2
policy update --id 1 --name my-policy --category my-category --impact HIGH --redaction First-1 --fields myfield1,myfield2
policy delete --id 1
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	//add subcommands
	cmd.AddCommand(newViewPolicyCommand())
	cmd.AddCommand(newCreatePolicyCommand())
	cmd.AddCommand(newUpdatePolicyCommand())
	cmd.AddCommand(newUpdatePolicyCommand())
	cmd.AddCommand(newDeletePolicyCommand())
	return cmd
}

func newViewPolicyCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:              "view",
		Short:            "View a policy",
		Example:          `policy view --name my-policy`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {


			policies, err := client.GetPolicies(); 
			
			if err != nil {
				golog.Errorf("Failed to retrieve policies. [%s]", err.Error())
				return err
			}

			for _, p := range policies {
				if p.Name == name {
					bite.PrintObject(cmd, p)
					return nil
				}
			}

			golog.Errorf("Policy [%s] does not exist", name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Policy name")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func newCreatePolicyCommand() *cobra.Command {
	var policy lenses.DataPolicyRequest
	var fields string

	cmd := &cobra.Command{
		Use:              "create",
		Short:            "Create a policy",
		Example:          `policy create --name my-policy --category my-category --impact HIGH --redaction First-1 --fields myfield1,myfield2`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			flags := bite.FlagPair{
						"name": policy.Name,
						"category": policy.Category,
						"redaction": policy.Obfuscation,
						"impact": policy.ImpactType,
						"fields": fields,
					}

			if err := bite.CheckRequiredFlags(cmd, flags); err != nil {
					return err
			}

			policy.Fields = strings.Split(fields, ",")

			if err := client.CreatePolicy(policy); err != nil {
				golog.Errorf("Failed to create policy [%s]. [%s]", policy.Name, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Policy [%s] created", policy.Name)
		},
	}

	cmd.Flags().StringVar(&policy.Name, "name", "", "Policy name")
	cmd.Flags().StringVar(&policy.Category, "category", "", "Policy category")
	cmd.Flags().StringVar(&policy.ImpactType, "impact", "", "Policy impact type")
	cmd.Flags().StringVar(&policy.Obfuscation, "redaction", "", "Policy redaction type")
	cmd.Flags().StringVar(&fields, "fields", "", "Schema fields, comma separated")
	bite.Prepend(cmd, bite.FileBind(&policy))
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func newUpdatePolicyCommand() *cobra.Command {
	var policy lenses.DataPolicyUpdateRequest
	var fields string

	cmd := &cobra.Command{
		Use:              "update",
		Short:            "Update a policy",
		Example:          `policy update --id 1 --name my-policy --category my-category --impact HIGH --redaction First-1 --fields myfield1,myfield2		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			flags := bite.FlagPair{
						"id":  policy.ID,
						"name": policy.Name,
						"category": policy.Category,
						"redaction": policy.Obfuscation,
						"impact": policy.ImpactType,
						"fields": fields,
					}

			if err := bite.CheckRequiredFlags(cmd, flags); err != nil {
					return err
			}

			policy.Fields = strings.Split(fields, ",")

			if err := client.UpdatePolicy(policy); err != nil {
				golog.Errorf("Failed to update policy [%s]. [%s]", policy.Name, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Policy [%s] updated", policy.Name)
		},
	}

	cmd.Flags().StringVar(&policy.ID, "id", "", "Policy id")
	cmd.Flags().StringVar(&policy.Name, "name", "", "Policy name")
	cmd.Flags().StringVar(&policy.Category, "category", "", "Policy category")
	cmd.Flags().StringVar(&policy.ImpactType, "impact", "", "Policy impact type")
	cmd.Flags().StringVar(&policy.Obfuscation, "redaction", "", "Policy redaction type")
	cmd.Flags().StringVar(&fields, "fields", "", "Schema fields, comma separated")
	bite.Prepend(cmd, bite.FileBind(&policy))
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func newDeletePolicyCommand() *cobra.Command {
	var id string

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete a policy",
		Example:          `policy delete --id`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"id":  id,}); err != nil {
					return err
			}


			if err := client.DeletePolicy(id); err != nil {
				golog.Errorf("Failed to delete policy [%s]. [%s]", id, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Policy [%s] deleted", id)
		},
	}

	cmd.Flags().StringVar(&id, "id", "", "Policy id")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func newGetPoliciesObfuscationCommand() *cobra.Command {
	
	cmd := &cobra.Command{
		Use:              "redactions",
		Short:            "List available redactions",
		Example:          `policies redactions`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

		
			r, err := client.GetPolicyObfuscation()
			
			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, r)
		},
	}

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func newGetPoliciesImpactTypesCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "impact-types",
		Short:            "List available impact-types",
		Example:          `policies impact-types`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

		
			r, err := client.GetPolicyImpacts()
			
			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, r)
		},
	}

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

