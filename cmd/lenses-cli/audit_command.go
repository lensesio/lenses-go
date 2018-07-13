package main

import (
	"github.com/landoop/lenses-go"

	"github.com/landoop/bite"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newGetAuditEntriesCommand())
}

func newGetAuditEntriesCommand() *cobra.Command {
	var sse bool

	cmd := &cobra.Command{
		Use:              "audits",
		Short:            "List the last buffered audit entries",
		Example:          `audits [--live]`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bite.FriendlyError(cmd, errResourceNotAccessibleMessage, "unable to access audit entries, user has no rights for this action")

			if sse {
				handler := func(entry lenses.AuditEntry) error {
					return bite.PrintObject(cmd, entry)
				}

				return client.GetAuditEntriesLive(handler)
			}

			entries, err := client.GetAuditEntries()
			if err != nil {
				return err
			}

			return bite.PrintObject(cmd, entries)
		},
	}

	cmd.Flags().BoolVar(&sse, "live", false, "--live")

	return cmd
}
