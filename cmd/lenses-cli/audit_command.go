package main

import (
	"github.com/landoop/lenses-go"
	"github.com/landoop/tableprinter"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newGetAuditEntriesCommand())
}

func newGetAuditEntriesCommand() *cobra.Command {
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
			bite.FriendlyError(cmd, errResourceNotAccessibleMessage, "unable to access audit entries, user has no rights for this action")

			withoutContentColumn := !bite.GetMachineFriendlyFlag(cmd) && !tableOnlyWithContent
			if sse {
				handler := func(entry lenses.AuditEntry) error {
					if withoutContentColumn {
						// entry.Content = nil, no need.
						newEntry := tableprinter.RemoveStructHeader(entry, "Content")
						return bite.PrintObject(cmd, newEntry)

					}
					return bite.PrintObject(cmd, entry)
				}

				return client.GetAuditEntriesLive(handler)
			}

			entries, err := client.GetAuditEntries()
			if err != nil {
				return err
			}

			if withoutContentColumn {
				// print each one without content,
				// bite is smart enough to see that it's the same type and it will append a row instead of a creating a new table,
				// although some further space on the "USER" header needed.
				for i := range entries {
					// entries[i].Content = nil
					newEntry := tableprinter.RemoveStructHeader(entries[i], "Content")
					if err = bite.PrintObject(cmd, newEntry); err != nil {
						return err
					}
				}

				return nil
			}

			return bite.PrintObject(cmd, entries)
		},
	}

	cmd.Flags().BoolVar(&sse, "live", false, "--live")
	cmd.Flags().BoolVar(&tableOnlyWithContent, "with-content", false, "--with-content add a table column to display the raw json content of the event action")
	return cmd
}
