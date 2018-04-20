package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func init() {
	rootCmd.AddCommand(newDocsCommand())
}

func newDocsCommand() *cobra.Command {
	return &cobra.Command{
		Use:                   "docs [directory]",
		Short:                 "Generate markdown commands documentation on cwd or any specific directory",
		Example:               "lenses-cli docs ./contents/",
		DisableFlagParsing:    true,
		DisableFlagsInUseLine: true,
		DisableSuggestions:    true,
		TraverseChildren:      false,
		Aliases:               []string{"doc"},
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 1 {
				return fmt.Errorf("directory argument is the only one valid argument")
			} else if len(args) == 0 {
				args = []string{"./"}
			}

			dir := args[0]

			// create directories if necessary.
			if err := os.MkdirAll(filepath.Dir(dir), os.FileMode(0755)); err != nil {
				return err
			}

			// generate markdown docs.
			return doc.GenMarkdownTree(rootCmd, dir)
		},
	}
}
