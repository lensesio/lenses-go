package connection

import (
	"github.com/lensesio/lenses-go/v5/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewAWSGroupCommand(gen genericConnectionClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "aws",
		Short:            "Manage Lenses AWS connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[api.ConfigurationObjectAWS]("AWS", gen, up, FlagMapperOpts{
		Descriptions: map[string]string{
			"AccessKeyId":     "Access key ID of an AWS IAM account.",
			"SecretAccessKey": "Secret access key of an AWS IAM account.",
			"Region":          "AWS region to connect to. If not provided, this is deferred to client configuration.",
			"SessionToken":    "Specifies the session token value that is required if you are using temporary security credentials that you retrieved directly from AWS STS operations.",
		},
	})...)
	return cmd
}
