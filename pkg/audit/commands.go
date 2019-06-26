package audit

import (
	"fmt"
	"strings"

	"github.com/landoop/bite"
	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/tableprinter"
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

	return cmd
}
