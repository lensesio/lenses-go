package main

import (
	"fmt"
	"os"
	// "net/url"
	// "sort"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/spf13/cobra"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"github.com/ghodss/yaml"
)

const (
	basePath = "landscape"
	appsPath = basePath + "/apps"
	processorPath = appsPath + "/sql/"
	connectorPath = appsPath + "/connectors/"
	dataEntitiesPath = basePath + "/data-entities"
	topicsPath = dataEntitiesPath + "/topics/"
	governancePath = basePath + "/governance"
	policiesPath = governancePath + "/policies/"
	controlListsPath = governancePath + "/control-lists/"
	quotasPath = governancePath + "/quotas/"
	monitoringPath = basePath + "/monitoring"
	alertsPath = monitoringPath + "/alerts/"
)

var resourceName, namespace, clusterName, stdout, branchName string

func init() {
	app.AddCommand(initRepoCommand())
	app.AddCommand(exportGroupCommand())
}

func createDirectory(directoryPath string) {
	//choose your permissions well
	pathErr := os.MkdirAll(directoryPath, 0777)

	//check if you need to panic, fallback or report
	if pathErr != nil {
		golog.Error(pathErr)
	}
}

func toYaml(o interface{}) ([]byte, error) {
	y, err := yaml.Marshal(o)
	return y, err
}

func toConsole(resource interface{}) {
	y, err := toYaml(resource)

	if err != nil {
		golog.Error(err)
	}

	fmt.Println(string(y))
}

func writeYaml(path string, resource interface{}) {

	y, err := toYaml(resource)

	if err != nil {
		golog.Error(err)
	}

	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)

	if err != nil {
		golog.Fatal(err)
	}
	defer file.Close()

	_, writeErr := file.Write(y)

	if writeErr != nil {
		golog.Fatal(writeErr)
	}
}

func writeProcessor(id string) {
	processor, err := client.ExportProcessor(id)

	if err != nil {
		golog.Error(err)
		golog.Error(err)
	}

	// write to output
	if stdout == "file" {
		identifier := processorPath + "processor-" + strings.Replace(id, ".", "-", len(id))
		writeYaml(identifier, processor)
	} else {
		println("\n---")
		toConsole(processor)
	}

}

func getAttachedTopics(id string) ([]lenses.TopicAsRequest, error) {
	var topics []lenses.TopicAsRequest

	extractedTopics, err := client.GetTopicExtract(id)

	if err != nil {
		return topics, err
	}

	for _, topicName := range extractedTopics {
		var tree = append(topicName.Decendants, topicName.Parents...)

		for _, t := range tree {
			if strings.HasPrefix(t, "TOPIC-") {
				var strippedTopicName = strings.Replace(t, "TOPIC-", "", len(t))
				topic, err := client.GetTopic(strippedTopicName)

				if err != nil {
					return topics, err
				}

				if err != nil {
					return topics, err
				}

				topics = append(topics, client.GetTopicAsRequest(topic))
			}
		}
	}

	return topics, nil
}

func createBranch(branchName string) error {

	dir, err := os.Getwd()
	
	if err != nil {
		golog.Fatal(err)
		return err
	}

	r, err := git.PlainOpen(dir)

	if err != nil {
		return err
	}

	w, err := r.Worktree()

	if err != nil {
		return err
	}

	branch := fmt.Sprintf("refs/heads/%s", branchName)
    b := plumbing.ReferenceName(branch)
	err = w.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: b})

	if err != nil {
		return err
	}

	golog.Info(fmt.Sprintf("Branch [%s] created", branchName))

	return nil
}

// writeConnectors writes the connectors to files as yaml
// If a clusterName is provided the connectors are filtered by clusterName
// If a name is provided the connectors are filtered by connector name
func writeConnectors(clusterName string, name string) error {
	clusters, err := client.GetConnectClusters()
	if err != nil {
		golog.Error(err)
		return err
	}

	for _, cluster := range clusters {

		connectorNames, err := client.GetConnectors(cluster.Name)
		if err != nil {
			golog.Error(err)
			return err
		}

		if clusterName != "" && clusterName != "*" {
			if cluster.Name != clusterName {
				continue
			}
		}

		for _, connectorName := range connectorNames {

			if name != "" {
				if connectorName != name {
					continue
				}
			}

			connector, err := client.GetConnector(cluster.Name, connectorName)
			request := lenses.ConnectorAsRequest{
				ClusterName: connector.ClusterName,
				Name:        connector.Name,
				Config:      connector.Config,
			}

			if err != nil {
				golog.Error(err)
				return err
			}

			//get topics
			topics, err := getAttachedTopics(fmt.Sprintf("%s:%s", connector.ClusterName, connector.Name))

			if err != nil {
				golog.Error(err)
				return err
			}

			if stdout == "file" {
				path := fmt.Sprintf("%sconnector-%s-%s.yaml", connectorPath, cluster.Name, connectorName)
				writeYaml(path, request)

				// write topics
				for _, topic := range topics {
					path := fmt.Sprintf("%stopic-%s.yaml", topicsPath, topic.TopicName)
					writeYaml(path, topic)
				}
			} else {
				println("\n---")
				toConsole(request)

				golog.Info("Attached Topics")
				for _, topic := range topics {
					println("\n---")
					toConsole(topic)
				}
			}
		}
	}
	return nil
}

func initRepoCommand() *cobra.Command {
	var gitURL, name string

	cmd := &cobra.Command{
		Use:              "init-repo",
		Short:            "Initialise a git repo to hold a landscape",
		Example:          `init-repo --git-url git@gitlab.com:landoop/demo-landscape.git`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			// create directory tree
			createDirectory(processorPath)
			createDirectory(connectorPath)
			createDirectory(topicsPath)
			createDirectory(policiesPath)
			createDirectory(controlListsPath)
			createDirectory(quotasPath)
			createDirectory(alertsPath)

			// initialise the git repo
			repo, err := git.PlainInit("", false)

			if err != nil {
				golog.Error("A repo already exists")
			}

			golog.Info("Landscape directory structure created")

			if gitURL != "" {
				golog.Info("Setting remote to [" + gitURL + "]")
				repo.CreateRemote(&config.RemoteConfig{
					Name: "origin",
					URLs: []string{gitURL},
				})
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&gitURL, "git-url", "", "--git-url=git@gitlab.com:landoop/demo-landscape.git Remote url to set for the repo, default is no remote")

	return cmd
}

func exportGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "export",
		Short:            "Export a landscape",
		Example:          `export [command] --resource-name my-resource`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			golog.Error("No subcommand provided")

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&branchName, "name", "", "--name=my-pipeline The name of the branch to create in git repo, if not set no branch is created")
	cmd.PersistentFlags().StringVar(&resourceName, "resource-name", "", "--resource-name=my-remote The resource name to export, defaults to all")
	cmd.PersistentFlags().StringVar(&clusterName, "clusterName", "", "--clusterName=clusterName select by cluster name, available only in CONNECT and KUBERNETES mode")
	cmd.PersistentFlags().StringVar(&namespace, "namespace", "", "--namespace=namespace select by namespace, available only in KUBERNETES mode")
	cmd.PersistentFlags().StringVar(&stdout, "stdout", "file", "--output=console output to the console or file. Default is file.")

	cmd.AddCommand(exportProcessorsCommand())
	cmd.AddCommand(exportConnectorsCommand())

	return cmd
}

func exportProcessorsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "processors",
		Short:            "export processors",
		Example:          `export processors --resource-name my-processor`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if branchName != "" {
				err := createBranch(branchName)

				if err != nil {
					golog.Error(err)
				}
			}

			mode, err := client.GetExecutionMode()
			if err != nil {
				return err
			}

			if mode == lenses.ExecutionModeInProcess {
				clusterName = "IN_PROC"
				namespace = "lenses"
			}

			if resourceName != "" {
				identifier, err := client.LookupProcessorIdentifier("", resourceName, clusterName, namespace)

				if err != nil {
					bite.FriendlyError(cmd, errResourceNotFoundMessage, fmt.Sprintf("Unable to find processor [%s] in namespace [%s] in cluster [%s]", resourceName, namespace, clusterName))
					return err
				}

				writeProcessor(identifier)
				return nil
			}

			// no name set to get all processors
			processors, err := client.GetProcessors()

			if err != nil {
				return err
			}

			for _, processor := range processors.Streams {

				if clusterName != "" && clusterName != processor.ClusterName {
					continue
				}

				if namespace != "" && namespace != processor.Namespace {
					continue
				}

				writeProcessor(processor.ID)
			}

			return nil
		},
	}

	return cmd
}

func exportConnectorsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "connectors",
		Short:            "export connectors",
		Example:          `export connectors --resource-name my-connector --clusterName Cluster1 If no clusterName is provided all connectors with the matching name are returned`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if branchName != "" {
				err := createBranch(branchName)

				if err != nil {
					golog.Error(err)
				}
			}
			writeConnectors(clusterName, resourceName)
			return nil
		},
	}

	return cmd
}
