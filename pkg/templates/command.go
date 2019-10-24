package templates

import (
	"strings"
	"errors"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	config "github.com/landoop/lenses-go/pkg/configs"
	cobra "github.com/spf13/cobra"
)

// NewTemplatesGroupCommand creates `templates` command
func NewTemplatesGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "templates",
		Short: `Manage templates`,
		Example: `
templates list
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.AddCommand(NewTemplatesListCommand())
	cmd.AddCommand(NewAddTemplatesCommand())

	return cmd
}

// NewTemplatesListCommand creates `templates list` command
func NewTemplatesListCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "list",
		Short: `List templates`,
		Example: `
templates list
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			connections, err := config.Client.ListTemplates()

			if err != nil {
				golog.Errorf("Failed to retrieve templates. [%s]", err.Error())
				return err
			}

			bite.PrintObject(cmd, connections)
			return nil
		},
	}

	bite.CanPrintJSON(cmd)
	return cmd
}

// NewAddTemplatesCommand creates `templates create` command
func NewAddTemplatesCommand() *cobra.Command {
	var tempale []api.TemplateCreatePayload

	cmd := &cobra.Command{
		Use:   "create",
		Short: `Create templates`,
		Example: `
template create ./my-template.yaml
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			return nil
		},
	}

	bite.Prepend(cmd, bite.FileBind(&tempale))
	return cmd
}

// NewDeleteTemplatesCommand creates `templates delete` command
func NewDeleteTemplatesCommand() *cobra.Command {
	var id	int

	cmd := &cobra.Command{
		Use:   "delete",
		Short: `Delete templates`,
		Example: `
templates delete --id 1
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Client.DeleteConnection(id); err != nil {
				golog.Errorf("Failed to delete template with id [%d]. [%s]", id, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Template has been successfully deleted.")
		},
	}

	cmd.Flags().IntVar(&id, "id", 0, "Numeric ID of the template")
	cmd.MarkFlagRequired("id")
	// Required for bite to send standard output to cmd execution buffer
	_ = bite.CanBeSilent(cmd)

	return cmd
}