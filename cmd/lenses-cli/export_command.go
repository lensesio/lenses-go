package main

import (
	"github.com/landoop/bite"
	"fmt"
	"os"
	// "net/url"
	// "sort"
	"strings"

	"github.com/landoop/lenses-go"

	"github.com/kataras/golog"
	"github.com/spf13/cobra"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"

	"gopkg.in/yaml.v2"
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

var resourceName, namespace, clusterName, branchName string

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

func writeYaml(basePath, fileName string, resource interface{}) error {

	y, err := toYaml(resource)

	if err != nil {
		golog.Error(err)
		return err
	}

	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		golog.Error(fmt.Sprintf("Directory [%s] does not exist", basePath))
		return err
	}

	path := fmt.Sprintf("%s/%s", basePath, fileName)

	file, err := os.OpenFile(
		path,
		os.O_WRONLY|os.O_TRUNC|os.O_CREATE,
		0666,
	)

	if err != nil {
		golog.Fatal(err)
		return err
	}
	defer file.Close()

	_, writeErr := file.Write(y)

	if writeErr != nil {
		golog.Fatal(writeErr)
		return writeErr
	}

	return nil
}


func writeProcessor(cmd *cobra.Command, cluster, namespace, name string) error {
	mode, err := client.GetExecutionMode()
	if err != nil {
		return err
	}

	if mode == lenses.ExecutionModeInProcess {
		clusterName = "IN_PROC"
		namespace = "lenses"
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

		if name != "" && name != processor.Name {
			continue
		}

		request := processor.ProcessorAsRequest()

		//get topics
		topics, err := getAttachedTopics(processor.ID)

		if err != nil {
			golog.Error(err)
			return err
		}


		if !bite.GetMachineFriendlyFlag(cmd) {

			var fileName string

			if mode == lenses.ExecutionModeInProcess {
				fileName = processor.Name
			} else if mode == lenses.ExecutionModeConnect {
				fileName = fmt.Sprintf("processor-%s-%s.yaml", processor.ClusterName, processor.Name)
			} else {
				fileName = fmt.Sprintf("processor-%s-%s-%s.yaml", processor.ClusterName, processor.Namespace, processor.Name)
			}

			// trim so the yaml is a multiline string
			request.SQL = strings.TrimSpace(request.SQL)

			err := writeYaml(processorPath, fileName, request)
			if err != nil {
				golog.Error(err)
				return err
			}

			// write topics
			for _, topic := range topics {
				fileName := fmt.Sprintf("topic-%s.yaml", topic.TopicName)
				err := writeYaml(topicsPath, fileName, topic)

				if err != nil {
					golog.Error(err)
					return err
				}
			}
		} else {
			println("\n---")
			bite.PrintJSON(cmd, request)

			golog.Info("Attached Topics")
			for _, topic := range topics {
				println("\n---")
				bite.PrintJSON(cmd, topic)
			}
		}
	}

	return nil
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

				overrides := getTopicConfigOverrides(topic.Config)
				topics = append(topics, topic.GetTopicAsRequest(overrides))
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
func writeConnectors(cmd *cobra.Command, clusterName string, name string) error {
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
			request := connector.ConnectorAsRequest()

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

			if !bite.GetMachineFriendlyFlag(cmd) {
				fileName := fmt.Sprintf("connector-%s-%s.yaml", cluster.Name, connectorName)
				err := writeYaml(connectorPath, fileName, request)
				if err != nil {
					golog.Error(err)
					return err
				}

				// write topics
				for _, topic := range topics {
					fileName := fmt.Sprintf("topic-%s.yaml", topic.TopicName)
					err := writeYaml(topicsPath, fileName, topic)
					if err != nil {
						golog.Error(err)
						return err
					}
				}
			} else {
				println("\n---")
				bite.PrintJSON(cmd, request)

				golog.Info("Attached Topics")
				for _, topic := range topics {
					println("\n---")
					bite.PrintJSON(cmd, topic)
				}
			}
		}
	}
	return nil
}

func writeTopics(topicName string) error {

	topics, err := client.GetTopics()

	if err != nil {
		return err
	}

	for _, topic := range topics {
		overrides := getTopicConfigOverrides(topic.Config)
		asRequest := topic.GetTopicAsRequest(overrides)
		fileName := fmt.Sprintf("topic-%s.yaml", asRequest.TopicName)
		writeYaml(topicsPath, fileName, asRequest)

		if topicName == asRequest.TopicName {
			break
		}
	}

	return nil 
}

func getTopicConfigOverrides(configs []lenses.KV) []lenses.KV {
	var overrides []lenses.KV
	for _, kv := range configs {

		if val, ok := kv["isDefault"]; ok {
			if (val.(bool) == false) {
				overrides = append(overrides, kv)
			}
		}
	}

	return overrides
}

// func writeQuotas(quoteType, quotaName string) error {

// 	quotas, err := client.GetQuotas()

// 	if err != nil {
// 		return err
// 	}

// 	for _, quota := range quotas {
// 		request := quota.GetQuotaAsRequest()
// 		path := fmt.Sprintf("%squota-%s-%s", quotasPath, quoteType, entityName)
// 		writeYaml(path, request)

// 		if quotaName != "" || quotaName != "*" {
// 			if quotaName == quota.EntityName {
// 				if quoteType == quota.EntityType {
// 					break
// 				}
// 			}
// 		}
// 	}

// 	return nil
// }

// func writeAlerts(alertName string) error {
// 	alerts, err := client.GetAlerts()

// 	if err != nil {
// 		return err
// 	}

// 	for _, alert := range alerts {
// 		request := alert.GetAlertAsRequest()
// 		path := fmt.Sprintf("%salert-%s", alertsPath, alertName)
// 		writeYaml(path, request)

// 		if alertName == alert.Name {
// 			break
// 		}
// 	}

// 	return nil
// }

// func writeACLs(name string) error {
// 	acls, err := client.GetACLs()

// 	if err != nil {
// 		return err
// 	}

// 	for _, acl := range acls {
// 		path := fmt.Sprintf("%sacl-%s", controlListsPath, name)
// 		writeYaml(path, acl)

// 		if name == acl.Name {
// 			break
// 		}
// 	}

// 	return nil
// }

func addGitSupport(gitURL string) error {
	repo, err := git.PlainOpen("")

	if err == nil {
		pwd, _ := os.Getwd()
		golog.Error(fmt.Sprintf("Git repo already exists in directory [%s]", pwd))
		return err
	}

	// initialise the git repo
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

	file.WriteString("/export")	

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

	golog.Info("Landscape directory structure created")

	if gitURL != "" {
		golog.Info("Setting remote to [" + gitURL + "]")
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{gitURL},
		})
	}

	return nil
}

func setupBranch(branchName string) error {
	if branchName != "" {
		err := createBranch(branchName)

		if err != nil {
			golog.Error(err)
			return err
		}
	}

	return nil
}

func initRepoCommand() *cobra.Command {
	var gitURL string
	var gitSupport bool

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

			if gitSupport {
				addGitSupport(gitURL)
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&gitSupport, "git", false, "--git Initialize a git repo")
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

	cmd.AddCommand(exportProcessorsCommand())
	cmd.AddCommand(exportConnectorsCommand())
	cmd.AddCommand(exportTopicsCommand())
	cmd.AddCommand(exportAlertsCommand())
	cmd.AddCommand(exportQuotasCommand())
	cmd.AddCommand(exportAclsCommand())
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
			writeProcessor(cmd, clusterName, namespace, resourceName)
			return nil
		},
	}

	cmd.Flags().StringVar(&resourceName, "resource-name", "", "--resource-name=my-processor The resource name to export, defaults to all")
	cmd.Flags().StringVar(&clusterName, "clusterName", "", "--clusterName=clusterName select by cluster name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&namespace, "namespace", "", "--namespace=namespace select by namespace, available only in KUBERNETES mode")
	return cmd
}

func exportTopicsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "topics",
		Short:            "export topics",
		Example:          `export topics --resource-name my-topic`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			writeTopics(resourceName)
			return nil
		},
	}

	cmd.Flags().StringVar(&resourceName, "resource-name", "", "--resource-name=my-topic The resource name to export, defaults to all")
	return cmd
}

func exportAlertsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "alerts",
		Short:            "export alerts",
		Example:          `export alerts --resource-name my-alert`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// writeAlerts(resourceName)
			return nil
		},
	}

	cmd.Flags().StringVar(&resourceName, "resource-name", "", "--resource-name=my-alert The resource name to export, defaults to all")
	return cmd
}

func exportAclsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "export acls",
		Example:          `export acls --resource-name my-acls`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// writeACLs(resourceName)
			return nil
		},
	}

	cmd.Flags().StringVar(&resourceName, "resource-name", "", "--resource-name=my-acl The resource name to export, defaults to all")
	return cmd
}

func exportQuotasCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "quotas",
		Short:            "export quotas",
		Example:          `export quota --resource-name my-quota`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			// writeQuotas(resourceName, "entity-type")
			return nil
		},
	}

	cmd.Flags().StringVar(&resourceName, "resource-name", "", "--resource-name=my-quota The resource name to export, defaults to all")
	cmd.Flags().StringVar(&resourceName, "entity-type", "", "--entity-type=my-quota-entity-type The quota entity type name to export, defaults to all")
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
			writeConnectors(cmd, clusterName, resourceName)
			return nil
		},
	}

	cmd.Flags().StringVar(&resourceName, "resource-name", "", "--resource-name=my-remote The resource name to export, defaults to all")
	cmd.Flags().StringVar(&clusterName, "clusterName", "", "--clusterName=clusterName select by cluster name, available only in CONNECT and KUBERNETES mode")
	return cmd
}
