package audit

import (
	"fmt"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/spf13/cobra"
)

// TODO AC-1458  - a parent command ?

//NewGetAuditChannelsCommand creates the `auditchannels` command
func NewGetAuditChannelsCommand() *cobra.Command {
	var (
		page         int
		pageSize     int
		sortField    string
		sortOrder    string
		templateName string
		channelName  string
		details      bool
	)

	cmd := &cobra.Command{
		Use:              "auditchannels",
		Short:            "Print the registered audit channels",
		Example:          `auditchannels --page=1 --pageSize=10 --sortField="name" --sortOrder="asc" --templateName="test" --channelName="slack" --details`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			auditChannelsPath := pkg.AuditChannelsPath

			if details {
				auditchannelsWithDetails, err := config.Client.GetChannelsWithDetails(auditChannelsPath, page, pageSize, sortField, sortOrder, templateName, channelName)
				if err != nil {
					return fmt.Errorf("failed to retrieve audits' channels. Error: [%s]", err.Error())
				}
				return bite.PrintObject(cmd, auditchannelsWithDetails.Values)
			}

			auditchannels, err := config.Client.GetChannels(auditChannelsPath, page, pageSize, sortField, sortOrder, templateName, channelName)
			if err != nil {
				return fmt.Errorf("failed to retrieve audits' channels. Error: [%s]", err.Error())
			}
			return bite.PrintObject(cmd, auditchannels.Values)
		},
	}

	cmd.Flags().IntVar(&page, "page", 1, "The page number to be fetched, must be greater than zero. Defaults to 1")
	cmd.Flags().IntVar(&pageSize, "pageSize", 10, "The amount of items to return in a single page, must be greater than zero.")
	cmd.Flags().StringVar(&sortField, "sortField", "", `The field to sort channel results by. Defaults to createdAt`)
	cmd.Flags().StringVar(&sortOrder, "sortOrder", "", `Choices: "asc" or "desc"`)
	cmd.Flags().StringVar(&templateName, "templateName", "", `Filter channels by template name.`)
	cmd.Flags().StringVar(&channelName, "channelName", "", `Filter channels with a name matching the supplied string (e.g. kafka-prd would match kafka-prd-pagerduty and kafka-prd-slack).`)
	cmd.Flags().BoolVar(&details, "details", false, `--details`)

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)

	// cmd.AddCommand(NewDeleteAuditChannelCommand())
	// cmd.AddCommand(NewCreateAuditChannelCommand())
	// cmd.AddCommand(NewUpdateAuditChannelCommand())

	return cmd
}
