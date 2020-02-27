package connection

import (
	"strings"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	cobra "github.com/spf13/cobra"
)

// NewConnectionGroupCommand creates `connection` command
func NewConnectionGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "connections",
		Short: `Manage Lenses external connections`,
		Example: `
connections
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			connections, err := config.Client.GetConnections()
			if err != nil {
				golog.Errorf("Failed to retrieve connections. [%s]", err.Error())
				return err
			}

			outputFlagValue := strings.ToUpper(bite.GetOutPutFlag(cmd))
			if outputFlagValue != "JSON" && outputFlagValue != "YAML" {
				bite.PrintInfo(cmd, "Info: use JSON or YAML output to get the complete object\n\n")
			}

			return bite.PrintObject(cmd, connections)
		},
	}

	cmd.AddCommand(NewConnectionGetCommand())
	cmd.AddCommand(NewConnectionCreateCommand())
	cmd.AddCommand(NewConnectionDeleteCommand())
	cmd.AddCommand(NewConnectionUpdateCommand())

	bite.CanPrintJSON(cmd)

	return cmd
}

// NewConnectionGetCommand creates `connections get` group command
func NewConnectionGetCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "get",
		Short: `Get Lenses connections`,
		Example: `
connections get --name connection-name
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			connection, err := config.Client.GetConnection(name)
			if err != nil {
				golog.Errorf("Failed to retrieve connections. [%s]", err.Error())
				return err
			}

			outputFlagValue := strings.ToUpper(bite.GetOutPutFlag(cmd))
			if outputFlagValue != "JSON" && outputFlagValue != "YAML" {
				bite.PrintInfo(cmd, "Info: use JSON or YAML output to get the complete object\n\n")
			}

			return bite.PrintObject(cmd, connection)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "connection name")
	cmd.MarkFlagRequired("name")

	bite.CanPrintJSON(cmd)

	return cmd
}

// NewConnectionCreateCommand creates `connections create` group command
func NewConnectionCreateCommand() *cobra.Command {
	var name, connectionConfig, templateName string
	var tags []string

	cmd := &cobra.Command{
		Use:   "create",
		Short: `Create a Lenses connection`,
		Example: `
connections create --name connection1 \
                   --tag t1 \
                   --template-name Cassandra \
                   --connection-config '[{"key":"port","value":["9042"]},{"key":"contact-points","value":["cassandra-host"]},{"key":"ssl-client-cert-auth","value":true}]'
                `,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Client.CreateConnection(name, templateName, connectionConfig, []api.ConnectionConfig{}, tags); err != nil {
				golog.Errorf("Failed to create Lenses connection. [%s]", err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Lenses connection has been successfully created.")
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the connection")
	cmd.Flags().StringVar(&templateName, "template-name", "", "Template connection name")
	cmd.Flags().StringVar(&connectionConfig, "connection-config", "", "configuration keys and values as json. Example: [{\"key\":\"port\",\"value\":[\"9042\"]}]")
	cmd.Flags().StringArrayVar(&tags, "tag", []string{}, "tag assigned to the connection, can be defined multiple times")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("template-name")
	cmd.MarkFlagRequired("connection-config")
	// Required for bite to send standard output to cmd execution buffer
	_ = bite.CanBeSilent(cmd)

	return cmd
}

// NewConnectionUpdateCommand creates `connections update` group command
func NewConnectionUpdateCommand() *cobra.Command {
	var name, newName, connectionConfig string
	var tags []string

	cmd := &cobra.Command{
		Use:   "update",
		Short: `Update a Lenses connection. Check the associated connection template for required configuration values.`,
		Example: `
connections update --name connection1 \
                   --tag t1 \
                   --connection-config '[{"key":"port","value":["444"]},{"key":"contact-points","value":["myhost"]},{"key":"ssl-client-cert-auth","value":true}]'
                `,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Client.UpdateConnection(name, newName, connectionConfig, []api.ConnectionConfig{}, tags); err != nil {
				golog.Errorf("Failed to update Lenses connection. [%s]", err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Lenses connection has been successfully updated.")
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the connection")
	cmd.Flags().StringVar(&connectionConfig, "connection-config", "", "configuration keys and values as json. Example: [{\"key\":\"port\",\"value\":[\"9042\"]}]")
	cmd.Flags().StringArrayVar(&tags, "tag", []string{}, "tag assigned to the connection, can be defined multiple times")
	cmd.MarkFlagRequired("name")
	cmd.MarkFlagRequired("connection-config")
	// Required for bite to send standard output to cmd execution buffer
	_ = bite.CanBeSilent(cmd)

	return cmd
}

// NewConnectionDeleteCommand creates `connections delete` group command
func NewConnectionDeleteCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: `Delete a Lenses connections`,
		Example: `
connections delete --name connection-name
                `,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Client.DeleteConnection(name); err != nil {
				golog.Errorf("Failed to delete connection. [%s]", err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Lenses connection has been successfully deleted.")
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "connection name")

	// Required for bite to send standard output to cmd execution buffer
	_ = bite.CanBeSilent(cmd)

	return cmd
}
