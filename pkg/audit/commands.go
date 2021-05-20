package audit

import (
	"fmt"
	"strings"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	"github.com/lensesio/tableprinter"
	"github.com/spf13/cobra"
)

//NewGetAuditEntriesCommand  creates the `audits` command
func NewGetAuditEntriesCommand() *cobra.Command {
	var (
		sse                  bool
		tableOnlyWithContent bool
	)

	cmd := &cobra.Command{
		Use:              "audits",
		Short:            "List the last buffered audit entries",
		Example:          `audits [--live] [--with-content]`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Audits entries are accessible for all roles atm.
			withoutContentColumn := strings.ToUpper(bite.GetOutPutFlag(cmd)) == "TABLE" && !tableOnlyWithContent
			if sse {
				handler := func(entry api.AuditEntry) error {
					if withoutContentColumn {
						// entry.Content = nil, no need.
						newEntry := tableprinter.RemoveStructHeader(entry, "Content")
						return bite.PrintObject(cmd, newEntry)

					}
					return bite.PrintObject(cmd, entry)
				}

				return config.Client.GetAuditEntriesLive(handler)
			}

			entries, err := config.Client.GetAuditEntries()
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				return nil
			}

			if withoutContentColumn {
				// print each one without content,
				// bite is smart enough to see that it's the same type and it will append a row instead of a creating a new table,
				// although some further space on the "USER" header needed.
				for i := range entries {
					// entries[i].Content = nil
					newEntry := tableprinter.RemoveStructHeader(entries[i], "Content")
					// show the length of types by overriding the type header struct(cached or not), printer don't really know how much they are in this time.
					// LINK:api.Entry.Type
					newEntry = tableprinter.SetStructHeader(newEntry, "Type", fmt.Sprintf("TYPE [%d]", len(entries)))
					if err = bite.PrintObject(cmd, newEntry); err != nil {
						return err
					}
				}

				return nil
			}

			return bite.PrintObject(cmd, entries)
		},
	}

	cmd.Flags().BoolVar(&sse, "live", false, "Subscribe to live audit feeds")
	cmd.Flags().BoolVar(&tableOnlyWithContent, "with-content", false, "Add a table column to display the raw json content of the event action")

	bite.CanPrintJSON(cmd)

	cmd.AddCommand(DeleteAuditEntriesCommand())

	return cmd
}

//DeleteAuditEntriesCommand  creates the `audits delete` command
func DeleteAuditEntriesCommand() *cobra.Command {
	var olderThanTimestamp int64

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete all the audit logs older than specified timestamp",
		Example:          "audits delete --timestamp=1621244454127",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"timestamp": olderThanTimestamp}); err != nil {
				return err
			}

			if err := config.Client.DeleteAuditEntries(olderThanTimestamp); err != nil {
				return fmt.Errorf("Failed to delete audit logs. [%s]", err.Error())
			}
			return bite.PrintInfo(cmd, "Audit logs older than timestamp: [%d] deleted.", olderThanTimestamp)
		},
	}

	cmd.Flags().Int64Var(&olderThanTimestamp, "timestamp", 0, "All the audit logs older than that timestamp will be removed.")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

// NewGetAuditChannelTemplatesCommand creates the `auditchannel-templates` sub-command
func NewGetAuditChannelTemplatesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auditchannel-templates",
		Short: "List audit channel templates",
		Example: `
# List all audit channel templates
auditchannel-templates

# Do a simple query using jq
auditchannel-templates --output=json | jq '.[] | select(.name =="Splunk")' 
`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			auditChannelTemplates, err := config.Client.GetAuditChannelTemplates()
			if err != nil {
				return fmt.Errorf("failed to retrieve audit channel templates. [%s]", err.Error())
			}

			outputFlagValue := strings.ToUpper(bite.GetOutPutFlag(cmd))
			if outputFlagValue != "JSON" && outputFlagValue != "YAML" {
				bite.PrintInfo(cmd, "Info: use JSON or YAML output to get the complete object\n\n")
			}

			return bite.PrintObject(cmd, auditChannelTemplates)
		},
	}

	bite.CanPrintJSON(cmd)

	return cmd
}
