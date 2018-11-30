package main

import (
	"github.com/landoop/bite"
	"fmt"
	"os"
	"strconv"
	"strings"
	"encoding/json"

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
	schemasPath = basePath + "/schemas"

	connectorClassKey = "connector.class"
	sqlConnectorClass = "com.landoop.connect.SQL"
)

var mode lenses.ExecutionMode
var toFile, dependents bool

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

func writeFile(basePath, fileName, format string, resource interface{}) error {
	if format == "YAML" {
		return writeYAML(basePath, fileName, resource)
	}

	return writeJSON(basePath, fileName, resource)
}

func write(basePath, fileName string, data []byte) error {

	if _, err := os.Stat(basePath); os.IsNotExist(err) {
		golog.Errorf("Directory [%s] does not exist, run --init-repo to initialize", basePath)
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

	_, writeErr := file.Write(data)

	if writeErr != nil {
		golog.Fatal(writeErr)
		return writeErr
	}

	return nil
}

func writeResourceToFile(cmd *cobra.Command, in interface{}, path string, fileName string) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	if toFile {
		if output == "YAML" || output == "JSON" {
			err := writeFile(path, fileName, output, in)

			if err != nil {
				golog.Error(err)
				return err
			}
		}
	} else {
		if (output == "TABLE") {
			output = "YAML"
		}
		cmd.Flag(bite.GetOutPutFlagKey()).Value.Set(output)
		bite.PrintObject(cmd, in)
	}

	return nil
}

func writeJSON(basePath, fileName string, resource interface{}) error {

	y, err := json.Marshal(resource)

	if err != nil {
		golog.Error(err)
		return err
	}

	write(basePath, fileName, y)

	return nil
}

func writeYAML(basePath, fileName string, resource interface{}) error {

	y, err := toYaml(resource)

	if err != nil {
		golog.Error(err)
		return err
	}

	write(basePath, fileName, y)
	return nil
}

func setExecutionMode() error {
	execMode, err := getExecutionMode()
	
	if err != nil {
		return err
	}

	mode = execMode
	return nil
}

func getExecutionMode() (lenses.ExecutionMode, error) {
	mode, err := client.GetExecutionMode()
	if err != nil {
		return mode, err
	}

	return mode, nil
}


func writeProcessor(cmd *cobra.Command, id, cluster, namespace, name string) error {

	if mode == lenses.ExecutionModeInProcess {
		cluster = "IN_PROC"
		namespace = "lenses"
	}

	// no name set to get all processors
	processors, err := client.GetProcessors()

	if err != nil {
		return err
	}

	for _, processor := range processors.Streams {

		if id != "" && id != processor.ID {
			continue
		} else {
			if cluster != "" && cluster != processor.ClusterName {
				continue
			}
	
			if namespace != "" && namespace != processor.Namespace {
				continue
			}
	
			if name != "" && name != processor.Name {
				continue
			}
		}

		request := processor.ProcessorAsRequest()

		output := strings.ToUpper(bite.GetOutPutFlag(cmd))
		if (output == "TABLE") {
			output = "YAML"
		}

		if toFile {

			var fileName string

			if mode == lenses.ExecutionModeInProcess {
				fileName = processor.Name
			} else if mode == lenses.ExecutionModeConnect {
				fileName = fmt.Sprintf("processor-%s-%s.%s", processor.ClusterName, processor.Name, strings.ToLower(output))
			} else {
				fileName = fmt.Sprintf("processor-%s-%s-%s.%s", processor.ClusterName, processor.Namespace, processor.Name, strings.ToLower(output))
			}

			// trim so the yaml is a multiline string
			request.SQL = strings.TrimSpace(request.SQL)
			request.SQL = strings.Replace(request.SQL, "\t", "  ", -1)
			request.SQL = strings.Replace(request.SQL, " \n", "\n", -1)

			if output == "YAML" || output == "JSON" {
				err := writeFile(processorPath, fileName, output, request)
				if err != nil {
					golog.Error(err)
					return err
				}
			}

			if dependents {
				handleDependents(cmd, processor.ID)
			}
		} else {
			cmd.Flag(bite.GetOutPutFlagKey()).Value.Set(output)
			bite.PrintObject(cmd, request)
			if dependents {
				handleDependents(cmd, processor.ID)
			}
		}
	}

	return nil
}

func getAttachedTopics(id string) ([]lenses.TopicAsRequest, error) {
	var topics []lenses.TopicAsRequest

	if dependents {
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
	}

	return topics, nil
}

func createBranch(cmd *cobra.Command, branchName string) error {

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

	bite.PrintInfo(cmd, "Branch [%s] created", branchName)

	return nil
}

func handleDependents(cmd *cobra.Command, id string) error {

	//get topics
	topics, err := getAttachedTopics(id)

	if err != nil {
		golog.Error(err)
		return err
	}

	var topicNames []string

	for _, t := range topics {
		topicNames = append(topicNames, t.Name)
	}

	if len(topics) == 0 && dependents {
		golog.Error(fmt.Sprintf("No topics found in the topology for processor [%s]", id))
	}

	// write topics
	writeTopicsAsRequest(cmd, topics)
	
	// get alert settings
	settings, err := getAlertSettings(cmd, topicNames)

	if err != nil {
		return err
	}

	writeAlertSettingsAsRequest(cmd, settings)

	//get acls
	acls, err := client.GetACLs()

	if err != nil {
		return err
	}

	var topicAcls []lenses.ACL

	for _, acl := range acls {
		if acl.ResourceType == lenses.ACLResourceTopic {
			for _, topicName := range topicNames {
				if acl.ResourceName == topicName {
					topicAcls = append(topicAcls, acl)
				}
			}
		}
	}
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("acls-topics.%s", strings.ToLower(output))
	writeResourceToFile(cmd, topicAcls, controlListsPath, fileName)

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

			if (connector.Config[connectorClassKey] == sqlConnectorClass) {
				continue
			}

			request := connector.ConnectorAsRequest()

			if err != nil {
				golog.Error(err)
				return err
			}

			output := strings.ToUpper(bite.GetOutPutFlag(cmd))
			fileName := fmt.Sprintf("connector-%s-%s.%s", cluster.Name, connectorName, strings.ToLower(output))

			if (output == "TABLE") {
				output = "YAML"
			}

			if toFile {
				if output == "YAML" || output == "JSON" {		
					err := writeFile(connectorPath, fileName, output, request)
					if err != nil {
						golog.Error(err)
						return err
					}

					handleDependents(cmd, fmt.Sprintf("%s:%s", connector.ClusterName, connector.Name))
				}
			
			} else {
				cmd.Flag(bite.GetOutPutFlagKey()).Value.Set(output)
				bite.PrintObject(cmd, request)
				if dependents {
					handleDependents(cmd, fmt.Sprintf("%s:%s", connector.ClusterName, connector.Name))
				}
			}
		}
	}
	return nil
}

func writeTopics(cmd *cobra.Command, topicName string) error {
	var requests []lenses.TopicAsRequest

	raw, err := client.GetTopics()

	if err != nil {
		return err
	}

	for _, topic := range raw {
		if topicName == "*" || topicName == "all" || topicName == topic.TopicName {
			overrides := getTopicConfigOverrides(topic.Config)
			requests = append(requests, topic.GetTopicAsRequest(overrides))
		}
	}

	wErr := writeTopicsAsRequest(cmd, requests)

	if wErr != nil {
		return wErr
	}
	
	return nil 
}

func writeTopicsAsRequest(cmd *cobra.Command, requests []lenses.TopicAsRequest) error {
	// write topics
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	for _, topic := range requests {

		fileName := fmt.Sprintf("topic-%s.%s", topic.Name, strings.ToLower(output))
		writeResourceToFile(cmd, topic, topicsPath, fileName)
	}

	return nil
}

func getTopicConfigOverrides(configs []lenses.KV) []lenses.KV {
	var overrides []lenses.KV
	for _, kv := range configs {

		if val, ok := kv["isDefault"]; ok {
			if (val.(bool) == false) {
				var name, value string

				if val, ok := kv["name"]; ok {
					name = val.(string)
				}

				if val, ok := kv["originalValue"]; ok {
					value = val.(string)
				}

				m := lenses.KV{}
				m[name] = value
				overrides = append(overrides, m)
			}
		}
	}

	return overrides
}

func writeQuotas(cmd *cobra.Command, quotaType string) error {

	quotas, err := client.GetQuotas()
	var requests []lenses.CreateQuotaPayload

	if err != nil {
		return err
	}

	for _, quota := range quotas {
		if quotaType == "ALL" || string(quota.EntityType) == quotaType {
			requests = append(requests, quota.GetQuotaAsRequest())
		}
	}

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("quotas-%s.%s", strings.Replace(strings.ToLower(quotaType), " ", "-", -1), strings.ToLower(output))
	writeResourceToFile(cmd, requests, quotasPath, fileName)
	
	return nil
}

func getAlertSettings(cmd *cobra.Command, topics []string) (AlertSettingConditionPayloads, error) {
	var alertSettings AlertSettingConditionPayloads
	var conditions []string
	

	settings, err := client.GetAlertSettings()

	if err != nil {
		return alertSettings, err
	}

	if len(settings.Categories.Consumers) == 0 {
		bite.PrintInfo(cmd, "No alert settings found ")
		return alertSettings, nil
	}

	consumerSettings := settings.Categories.Consumers
	
	for _, setting := range consumerSettings {
		for _, condition := range setting.Conditions {
			if len(topics) == 0 {
				conditions = append(conditions, condition)
				continue
			}

			// filter by topic name
			for _, topic := range topics {
				if (strings.Contains(condition, fmt.Sprintf("topic %s", topic))) {
					conditions = append(conditions, condition)
				}
			}
		}
	}

	if len(conditions) == 0 {
		bite.PrintInfo(cmd, "No consumer conditions found ")
		return alertSettings, nil
	}

	return AlertSettingConditionPayloads{AlertID: 2000, Conditions: conditions}, nil
}

func writeAlertSetting(cmd *cobra.Command) (error) {

	var topics []string
	settings, err := getAlertSettings(cmd, topics)

	if err != nil {
		return err
	}
	
	writeAlertSettingsAsRequest(cmd, settings)

	return nil
}

func writeAlertSettingsAsRequest(cmd *cobra.Command, settings AlertSettingConditionPayloads) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("alert-setting.%s", strings.ToLower(output))
	writeResourceToFile(cmd, settings, alertsPath, fileName)
	return nil
}

func writeACLs(cmd *cobra.Command, resourceType string, resourceName string) error {

	var acls []lenses.ACL
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("acls-%s.%s", strings.ToLower(resourceType) ,strings.ToLower(output))

	raw, err := client.GetACLs()

	if err != nil {
		return err
	}

	if resourceType == string(lenses.ACLResourceAny) {
		acls = raw
	} else {
		for _, acl := range raw {
			if string(acl.ResourceType) == resourceType || 
				resourceType == "*" || 
					resourceType == "all" {

				if resourceName == "*" || 
					resourceName == "all" || 
						acl.ResourceName == resourceName {
					acls = append(acls, acl)
				}
			} 
		}
	}

	writeResourceToFile(cmd, acls, controlListsPath, fileName) 

	return nil
}

func writeSchemas(cmd *cobra.Command) error {

	subjects, err := client.GetSubjects()

	if err != nil {
		return err
	}

	for _, subject := range subjects {
		writeSchema(cmd, subject, 0) 
	}

	return nil
}

func writeSchema(cmd *cobra.Command, name string, version int) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	var schema lenses.Schema
	var err error

	if version != 0 {
		schema, err = client.GetSchemaAtVersion(name, version)
	} else {
		schema, err = client.GetLatestSchema(name)
	}

	if err != nil {
		return err
	}

	request := client.GetSchemaAsRequest(schema)
	fileName := fmt.Sprintf("schema-%s.%s", strings.ToLower(name) ,strings.ToLower(output))
	writeResourceToFile(cmd, request, schemasPath, fileName) 

	return nil
}

func addGitSupport(cmd *cobra.Command, gitURL string) error {
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

	bite.PrintInfo(cmd, "Landscape directory structure created")

	if gitURL != "" {
		bite.PrintInfo(cmd, "Setting remote to [" + gitURL + "]")
		repo.CreateRemote(&config.RemoteConfig{
			Name: "origin",
			URLs: []string{gitURL},
		})
	}

	return nil
}

func setupBranch(cmd *cobra.Command, branchName string) error {
	if branchName != "" {
		err := createBranch(cmd, branchName)

		if err != nil {
			golog.Error(err)
			return err
		}
	}

	return nil
}

func checkFileFlags(cmd *cobra.Command) {

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	if toFile {
		if (output == "TABLE") {
			if !bite.HasSilentFlag(cmd) {
				bite.PrintInfo(cmd, "Defaulting to yaml format")
			}
			cmd.Flag(bite.GetOutPutFlagKey()).Value.Set("yaml")
			output = "YAML"
		}
	
		if (output != "JSON" && output != "YAML") {
			golog.Fatalf("Unsupported output format [%s]. Output type must be json or yaml for export", bite.GetOutPutFlag(cmd))
			return 
		}

		return
	}

	cmd.Flag(bite.GetOutPutFlagKey()).Value.Set(output)
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
				addGitSupport(cmd, gitURL)
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
			cmd.Help()

			return nil
		},
	}

	cmd.PersistentFlags().BoolVar(&toFile, "to-file", false, "--to-file Exports to files")
	cmd.PersistentFlags().BoolVar(&dependents, "dependents", false, "--dependents Extract dependencies, topics, acls, quotas, alerts" )
	cmd.AddCommand(exportProcessorsCommand())
	cmd.AddCommand(exportConnectorsCommand())
	cmd.AddCommand(exportTopicsCommand())
	cmd.AddCommand(exportAlertsCommand())
	cmd.AddCommand(exportQuotasCommand())
	cmd.AddCommand(exportAclsCommand())
	cmd.AddCommand(exportAllCommand())
	return cmd
}

func exportProcessorsCommand() *cobra.Command {
	var name, cluster, namespace, id string

	cmd := &cobra.Command{
		Use:              "processors",
		Short:            "export processors",
		Example:          `export processors --resource-name my-processor`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setExecutionMode()
			checkFileFlags(cmd)
			writeProcessor(cmd, id, cluster, namespace, name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "", "--resource-name=my-processor The resource name to export, defaults to all")
	cmd.Flags().StringVar(&cluster, "cluster-name", "", "--cluster-name=clusterName select by cluster name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&namespace, "namespace", "", "--namespace=namespace select by namespace, available only in KUBERNETES mode")
	cmd.Flags().StringVar(&id, "id", "", "--id=myid id of the processor")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func exportTopicsCommand() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:              "topics",
		Short:            "export topics",
		Example:          `export topics --resource-name my-topic`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			writeTopics(cmd, name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "*", "--resource-name=my-topic The resource name to export, defaults to all")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func exportAlertsCommand() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:              "alert-settings",
		Short:            "export alert-settings",
		Example:          `export alert-settings`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			writeAlertSetting(cmd)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "", "--resource-name=my-alert The resource name to export, defaults to all")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func exportAclsCommand() *cobra.Command {
	var resourceType, name string

	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "export acls",
		Example:          `export acls --resource-type CLUSTER`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			rt := strings.ToUpper(resourceType)

			if rt != string(lenses.ACLResourceCluster) &&
				rt != string(lenses.ACLResourceGroup) && 
					rt != string(lenses.ACLResourceTopic) && 
						rt != string(lenses.ACLResourceTransactionalID) &&
							rt != string(lenses.ACLResourceDelegationToken) && 
								rt != string(lenses.ACLResourceAny) &&
									rt != "*" &&
										rt != "all" {
				golog.Errorf("Unsupported resource type [%s]", rt)
				cmd.Help()
				return nil
			}

			writeACLs(cmd, rt, strings.ToLower(name))
			return nil
		},
	}

	cmd.Flags().StringVar(&resourceType, "resource-type", "*", "--resource-type=TOPIC The ACL resource type to export, Cluster, Topic, Group, TransactionalId, DelegationToken or Any, defaults to any")
	cmd.Flags().StringVar(&name, "resource-name", "*", "--resource-name=my-topic The ACL resource name to export, e.g. topic or consumer group, defaults to all")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func exportQuotasCommand() *cobra.Command {
	quotaType := "ALL"

	cmd := &cobra.Command{
		Use:              "quotas",
		Short:            "export quotas",
		Example:          `export quota --quota-type="CLIENTS DEFAULT" Quota type USER, USERCLIENT, USERS DEFAULT, CLIENT or CLIENTS DEFAULT`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			var qtype = strings.ToUpper(quotaType)

			if (qtype != "*" &&
				qtype != "ALL" &&
				qtype != string(lenses.QuotaEntityUser) && 
					qtype != string(lenses.QuotaEntityUserClient) && 
						qtype != string(lenses.QuotaEntityUsersDefault) && 
							qtype != string(lenses.QuotaEntityClient) && 
								qtype != string(lenses.QuotaEntityClientsDefault)) {
				golog.Errorf("Unsupported quota type [%s]", qtype)
				cmd.Help()
				return nil
			}

			writeQuotas(cmd, qtype)
			return nil
		},
	}

	cmd.Flags().StringVar(&quotaType, "quota-type", "*", "--quota-type='CLIENTDEFAULT' Quota type USER, USERCLIENT, USERS DEFAULT, CLIENT or CLIENT DEFAULT, defaults to all")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func exportConnectorsCommand() *cobra.Command {
	var name, cluster string

	cmd := &cobra.Command{
		Use:              "connectors",
		Short:            "export connectors",
		Example:          `export connectors --resource-name my-connector --cluster-name Cluster1 If no clusterName is provided all connectors with the matching name are returned`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setExecutionMode()
			checkFileFlags(cmd)
			writeConnectors(cmd, cluster, name)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "", "--resource-name=my-remote The resource name to export, defaults to all")
	cmd.Flags().StringVar(&cluster, "cluster-name", "", "--cluster-name=clusterName select by cluster name, available only in CONNECT and KUBERNETES mode")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func exportSchemasCommand() *cobra.Command {
	var name, version string

	cmd := &cobra.Command{
		Use:              "schemas",
		Short:            "export schemas",
		Example:          `export schemas --resource-name my-schema-value --version 1. If no name is supplied the latest versions of all schemas are exported`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			versionInt, err := strconv.Atoi(version)
			if err != nil {
				golog.Errorf("Version [%s] is not at integer", version)
				return err
			}
			
			if name == "" {
				writeSchema(cmd, name, versionInt)
				return nil
			} 

			writeSchemas(cmd)
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "", "--resource-name=my-schema The schema to export. Both the key schema and value schema are exported")
	cmd.Flags().StringVar(&version, "version", "0", "--version=1 The schema version to export.")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func exportAllCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "landscape",
		Short:            "export landscape",
		Example:          `export landscape Export processors, connectors, topics, quotas, acls, schemas and alert-settings`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setExecutionMode()
			checkFileFlags(cmd)
			writeProcessor(cmd, "", "", "", "")
			writeConnectors(cmd, "", "")
			writeTopics(cmd, "")
			writeQuotas(cmd, "*")
			writeAlertSetting(cmd)
			writeACLs(cmd, "*", "*")
			writeSchemas(cmd)
			return nil
		},
	}

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}
