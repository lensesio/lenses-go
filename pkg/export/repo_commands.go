package export

import (
	"fmt"
	"os"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
)

//InitRepoCommand creates the `init-repo` command
func InitRepoCommand() *cobra.Command {
	var gitURL string
	var gitSupport bool

	cmd := &cobra.Command{
		Use:              "init-repo",
		Short:            "Initialise a git repo to hold a landscape",
		Example:          `init-repo --git-url git@gitlab.com:landoop/demo-landscape.git`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if gitSupport {
				if err := addGitSupport(cmd, gitURL); err != nil {
					return err
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&landscapeDir, "name", "", "Directory name to repo in")
	cmd.Flags().BoolVar(&gitSupport, "git", false, "Initialize a git repo")
	cmd.Flags().StringVar(&gitURL, "git-url", "", "-Remote url to set for the repo")
	cmd.MarkFlagRequired("name")

	return cmd
}

func addGitSupport(cmd *cobra.Command, gitURL string) error {
	repo, err := git.PlainOpen("")

	if err == nil {
		pwd, _ := os.Getwd()
		golog.Error(fmt.Sprintf("Git repo already exists in directory [%s]", pwd))
		return err
	}

	// initialise the git
	repo, initErr := git.PlainInit("", false)

	if initErr != nil {
		golog.Error("A repo already exists")
	}

	file, err := os.OpenFile(
		".gitignore",
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)

	if err != nil {
		golog.Fatal(err)
	}
	defer file.Close()

	readme, err := os.OpenFile(
		"README.md",
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)

	if err != nil {
		golog.Fatal(err)
	}
	defer readme.Close()

	// write readme
	readme.WriteString(`# Lenses Landscape

This repo contains Lenses landscape resource descriptions described in yaml files
	`)

	wt, err := repo.Worktree()

	if err != nil {
		return err
	}

	wt.Add(".gitignore")
	wt.Add("landscape")
	wt.Add("README.md")

	bite.PrintInfo(cmd, "Landscape directory structure created")

	if gitURL != "" {
		bite.PrintInfo(cmd, "Setting remote to ["+gitURL+"]")
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{gitURL},
		})
	}

	return nil
}
