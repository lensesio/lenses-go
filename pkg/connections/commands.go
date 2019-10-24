package connections

import (
	"strings"
	"errors"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	config "github.com/landoop/lenses-go/pkg/configs"
	cobra "github.com/spf13/cobra"
)

// NewConnectionsGroupCommand creates `connections` command
func NewConnectionsGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "connections",
		Short: `Manage connections`,
		Example: `
connections list
connections create --name my-con --tags x,y,z --credential-id 1 --template-id 1 --config k1=v1,k2=v2,k3=v3
connections delete --id 1
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.AddCommand(NewConnectionsListCommand())
	cmd.AddCommand(NewAddConnectionsCommand())
	cmd.AddCommand(NewDeleteConnectionsCommand())

	return cmd
}

// NewConnectionsListCommand creates `connections list` command
func NewConnectionsListCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "list",
		Short: `List connections`,
		Example: `
connections list
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			connections, err := config.Client.ListConnections()

			if err != nil {
				golog.Errorf("Failed to retrieve connections. [%s]", err.Error())
				return err
			}

			bite.PrintObject(cmd, connections)
			return nil
		},
	}

	bite.CanPrintJSON(cmd)
	return cmd
}

// NewAddConnectionsCommand creates `connections create` command
func NewAddConnectionsCommand() *cobra.Command {
	var credentialID, templateID	int
	var name, tags, configKeys 		string


	cmd := &cobra.Command{
		Use:   "create",
		Short: `Create connections`,
		Example: `
connections create --name my-con --tags x,y,z --credential-id 1 --template-id 1 --config k1=v1,k2=v2,k3=v3
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			credentials, err := config.Client.ListCredentials()

			if err != nil {
				golog.Errorf("Failed to verify credential [%d]. [%s]", credentialID, err.Error())
				return err
			}

			for _, cred := range credentials {
				if cred.ID == credentialID {
			
					// create the connection instance 
					err := config.Client.CreateConnection(name, strings.Split(tags, ","), credentialID, templateID, strings.Split(configKeys, ","))
					if err != nil {
						golog.Errorf("Failed to create connection [%s]. [%s]", name, err.Error())
						return err
					}

					bite.PrintInfo(cmd, "Connection [%s] successfully created.", name)
					return nil
				}
			}

			golog.Errorf("Failed to create connection [%s]. CredentialID [%d] not found", name, credentialID)

			return errors.New("Create connection failed")
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Name of the connection")
	cmd.Flags().StringVar(&tags, "tags", "", "Comma separated list of tags")
	cmd.Flags().IntVar(&credentialID, "credential-id", -1, "Id of the credential to use to store the connection secrets")
	cmd.Flags().IntVar(&templateID, "template-id", -1, "Id of the connection template")
	cmd.Flags().StringVar(&configKeys, "configs", "", "Comma separated list of config key/value pairs")

	return cmd
}

// NewDeleteConnectionsCommand creates `connections delete` command
func NewDeleteConnectionsCommand() *cobra.Command {
	var id	int

	cmd := &cobra.Command{
		Use:   "delete",
		Short: `Delete connections`,
		Example: `
connections delete --id 1
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := config.Client.DeleteConnection(id); err != nil {
				golog.Errorf("Failed to delete connection with id [%d]. [%s]", id, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Connection has been successfully deleted.")
		},
	}

	cmd.Flags().IntVar(&id, "id", 0, "Numeric ID of the connection")
	cmd.MarkFlagRequired("id")
	// Required for bite to send standard output to cmd execution buffer
	_ = bite.CanBeSilent(cmd)

	return cmd
}
