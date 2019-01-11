package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/landoop/bite"

	"github.com/kataras/golog"
	"github.com/landoop/lenses-go"
	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/yaml.v2"
)

const (
	connectorClassKey = "connector.class"
	sqlConnectorClass = "com.landoop.connect.SQL"
)

var mode lenses.ExecutionMode
var dependents bool
var landscapeDir string
var systemTopicExclusions = []string{
	"connect-configs",
	"connect-offsets",
	"connect-status",
	"connect-statuses",
	"_schemas",
	"__consumer_offsets",
	"_kafka_lenses_",
	"lsql_",
	"__transaction_state",
	"__topology",
	"__topology__metrics",
	"_connect-configs",
	"_connect-status",
	"_connect-offsets",
	"_lenses_",
}

var topicExclusions string
var prefix string

func init() {
	app.AddCommand(initRepoCommand())
	app.AddCommand(exportGroupCommand())
}

func createDirectory(directoryPath string) error {
	return os.MkdirAll(directoryPath, 0777)
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

	dir := fmt.Sprintf("%s/%s", landscapeDir, basePath)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := createDirectory(dir); err != nil {
			return err
		}
	}

	path := fmt.Sprintf("%s/%s", dir, fileName)

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

func writeJSON(basePath, fileName string, resource interface{}) error {

	y, err := json.Marshal(resource)

	if err != nil {
		return err
	}

	return write(basePath, fileName, y)
}

func writeYAML(basePath, fileName string, resource interface{}) error {

	y, err := toYaml(resource)

	if err != nil {
		return err
	}

	return write(basePath, fileName, y)
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

func writeProcessors(cmd *cobra.Command, id, cluster, namespace, name string) error {

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

			if prefix != "" && !strings.HasPrefix(processor.Name, prefix) {
				continue
			}
		}

		request := processor.ProcessorAsRequest()

		output := strings.ToUpper(bite.GetOutPutFlag(cmd))

		if output == "TABLE" {
			output = "YAML"
		}

		var fileName string

		if mode == lenses.ExecutionModeInProcess {
			fileName = fmt.Sprintf("processor-%s.%s", strings.ToLower(processor.Name), strings.ToLower(output))
		} else if mode == lenses.ExecutionModeConnect {
			fileName = fmt.Sprintf("processor-%s-%s.%s", strings.ToLower(processor.ClusterName), strings.ToLower(processor.Name), strings.ToLower(output))
		} else {
			fileName = fmt.Sprintf("processor-%s-%s-%s.%s", strings.ToLower(processor.ClusterName), strings.ToLower(processor.Namespace), strings.ToLower(processor.Name), strings.ToLower(output))
		}

		// trim so the yaml is a multiline string
		request.SQL = strings.TrimSpace(request.SQL)
		request.SQL = strings.Replace(request.SQL, "\t", "  ", -1)
		request.SQL = strings.Replace(request.SQL, " \n", "\n", -1)

		if err := writeFile(sqlPath, fileName, output, request); err != nil {
			return err
		}

		if dependents {
			handleDependents(cmd, processor.ID)
		}
	}

	return nil
}

func getAttachedTopics(id string) ([]lenses.CreateTopicPayload, error) {
	var topics []lenses.CreateTopicPayload

	if dependents {
		extractedTopics, err := client.GetTopicExtract(id)

		if err != nil {
			return topics, err
		}

		for _, topicName := range extractedTopics {
			var tree = append(topicName.Descendants, topicName.Parents...)

			for _, t := range tree {
				if strings.HasPrefix(t, "TOPIC-") {
					var strippedTopicName = strings.Replace(t, "TOPIC-", "", len(t))
					topic, err := client.GetTopic(strippedTopicName)

					if err != nil {
						return topics, err
					}

					overrides := getTopicConfigOverrides(topic.Configs)
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
	if err = w.Checkout(&git.CheckoutOptions{Create: true, Force: false, Branch: b}); err != nil {
		return err
	}

	bite.PrintInfo(cmd, "Branch [%s] created", branchName)

	return nil
}

func handleDependents(cmd *cobra.Command, id string) error {

	//get topics
	topics, err := getAttachedTopics(id)

	if err != nil {
		return err
	}

	var topicNames []string

	for _, t := range topics {
		topicNames = append(topicNames, t.TopicName)
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
	fileName := fmt.Sprintf("acls-%s.%s", "all", strings.ToLower(output))
	return writeFile(aclsPath, fileName, output, topicAcls);
}

// writeConnectors writes the connectors to files as yaml
// If a clusterName is provided the connectors are filtered by clusterName
// If a name is provided the connectors are filtered by connector name
func writeConnectors(cmd *cobra.Command, clusterName string, name string) error {
	clusters, err := client.GetConnectClusters()

	if err != nil {
		return err
	}

	for _, cluster := range clusters {

		connectorNames, err := client.GetConnectors(cluster.Name)
		if err != nil {
			golog.Error(err)
			return err
		}

		if clusterName != "" && cluster.Name != clusterName {
			continue
		}

		for _, connectorName := range connectorNames {

			if name != "" && connectorName != name {
				continue
			}

			if prefix != "" && !strings.HasPrefix(connectorName, prefix) {
				continue
			}

			connector, err := client.GetConnector(cluster.Name, connectorName)

			if connector.Config[connectorClassKey] == sqlConnectorClass {
				continue
			}

			request := connector.ConnectorAsRequest()

			if err != nil {
				return err
			}

			output := strings.ToUpper(bite.GetOutPutFlag(cmd))
			fileName := fmt.Sprintf("connector-%s-%s.%s", strings.ToLower(cluster.Name), strings.ToLower(connectorName), strings.ToLower(output))

			if output == "TABLE" {
				output = "YAML"
			}

			golog.Debugf("Exporting connector [%s.%s] to [%s%s]", cluster.Name, connectorName, landscapeDir, fileName)
			if err := writeFile(connectorsPath, fileName, output, request); err != nil {
				return err
			}

			if dependents {
				handleDependents(cmd, fmt.Sprintf("%s:%s", connector.ClusterName, connector.Name))
			}
		}
	}
	return nil
}

func writeTopics(cmd *cobra.Command, topicName string) error {
	var requests []lenses.CreateTopicPayload

	raw, err := client.GetTopics()

	if err != nil {
		return err
	}

	for _, topic := range raw {

		// don't export control topics
		excluded := false
		for _, exclude := range systemTopicExclusions {
			if strings.HasPrefix(topic.TopicName, exclude) ||
				strings.Contains(topic.TopicName, "KSTREAM-") ||
				strings.Contains(topic.TopicName, "_agg_") ||
				strings.Contains(topic.TopicName, "_sql_store_") {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		// exclude any user defined
		excluded = false
		for _, exclude := range strings.Split(topicExclusions, ",") {
			if topic.TopicName == exclude {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		if prefix != "" && !strings.HasPrefix(topic.TopicName, prefix) {
			continue
		}

		if topicName != "" && topicName == topic.TopicName {
			overrides := getTopicConfigOverrides(topic.Configs)
			requests = append(requests, topic.GetTopicAsRequest(overrides))
			break
		}

		overrides := getTopicConfigOverrides(topic.Configs)
		requests = append(requests, topic.GetTopicAsRequest(overrides))
	}

	return writeTopicsAsRequest(cmd, requests)
}

func writeTopicsAsRequest(cmd *cobra.Command, requests []lenses.CreateTopicPayload) error {
	// write topics
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	for _, topic := range requests {

		fileName := fmt.Sprintf("topic-%s.%s", strings.ToLower(topic.TopicName), strings.ToLower(output))

		if err := writeFile(topicsPath, fileName, output, topic); err != nil {
			return err
		}
	}

	return nil
}

func getTopicConfigOverrides(configs []lenses.KV) lenses.KV {
	overrides := make(lenses.KV)

	for _, kv := range configs {
		if val, ok := kv["isDefault"]; ok {
			if val.(bool) == false {
				var name, value string

				if val, ok := kv["name"]; ok {
					name = val.(string)
				}

				if val, ok := kv["originalValue"]; ok {
					value = val.(string)
				}
				overrides[name] = value
			}
		}
	}

	return overrides
}

func writeQuotas(cmd *cobra.Command) error {

	quotas, err := client.GetQuotas()

	if err != nil {
		return err
	}

	var requests []lenses.CreateQuotaPayload
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("quotas.%s", strings.ToLower(output))

	for _, q := range quotas {
		requests = append(requests, q.GetQuotaAsRequest())
	}

	if err := writeFile(quotasPath, fileName, output, requests); err != nil {
		return err
	}

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
				if strings.Contains(condition, fmt.Sprintf("topic %s", topic)) {
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

func writeAlertSetting(cmd *cobra.Command) error {

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

	return writeFile(alertSettingsPath, fileName, output, settings)
}

func writeACLs(cmd *cobra.Command) error {

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("acls.%s", strings.ToLower(output))

	acls, err := client.GetACLs()

	if err != nil {
		return err
	}

	return writeFile(aclsPath, fileName, output, acls)
}

func writeSchemas(cmd *cobra.Command) error {

	subjects, err := client.GetSubjects()

	if err != nil {
		return err
	}

	for _, subject := range subjects {
		if prefix != "" && !strings.HasPrefix(subject, prefix) {
			continue
		}

		// don't export control topics
		excluded := false
		for _, exclude := range systemTopicExclusions {
			if strings.HasPrefix(subject, exclude) ||
				strings.Contains(subject, "KSTREAM-") ||
				strings.Contains(subject, "_agg_") ||
				strings.Contains(subject, "_sql_store_") {
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		return writeSchema(cmd, subject, 0)
	}

	return nil
}

func prettyPrint(b []byte) ([]byte, error) {
	var out bytes.Buffer
	err := json.Indent(&out, b, "", "  ")
	return out.Bytes(), err
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

	pretty, _ := prettyPrint([]byte(schema.AvroSchema))

	schema.AvroSchema = string(pretty)
	schema.AvroSchema = strings.TrimSpace(schema.AvroSchema)
	schema.AvroSchema = strings.Replace(schema.AvroSchema, "\t", "  ", -1)
	schema.AvroSchema = strings.Replace(schema.AvroSchema, " \n", "\n", -1)

	if err != nil {
		return err
	}

	request := client.GetSchemaAsRequest(schema)
	fileName := fmt.Sprintf("schema-%s.%s", strings.ToLower(name), strings.ToLower(output))
	return writeFile(schemasPath, fileName, output, request)
}

func writePolicies(cmd *cobra.Command, name string, ID string) error {
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	if ID != "" {
		policy, err := client.GetPolicy(ID)
		if err != nil {
			return err
		}

		fileName := fmt.Sprintf("policies-%s.%s", strings.ToLower(policy.Name), strings.ToLower(output))
		request := client.PolicyAsRequest(policy)
		return writeFile(policiesPath, fileName, output, request)
	}

	policies, err := client.GetPolicies()
	if err != nil {
		return err
	}

	for _, policy := range policies {
		fileName := fmt.Sprintf("policies-%s.%s", strings.ToLower(policy.Name), strings.ToLower(output))
		if name != "" && policy.Name == name {
			return writeFile(policiesPath, fileName, output, policy)
		}

		return writeFile(policiesPath, fileName, output, policy)
	}

	return nil
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

func setupBranch(cmd *cobra.Command, branchName string) error {
	if branchName != "" {
		if err := createBranch(cmd, branchName); err != nil {
			golog.Error(err)
			return err
		}
	}

	return nil
}

func checkFileFlags(cmd *cobra.Command) {

	output := strings.ToUpper(bite.GetOutPutFlag(cmd))

	if output == "TABLE" {
		output = "YAML"
	}

	if output != "JSON" && output != "YAML" {
		golog.Fatalf("Unsupported output format [%s]. Output type must be json or yaml for export", bite.GetOutPutFlag(cmd))
		return
	}

	cmd.Flag(bite.GetOutPutFlagKey()).Value.Set(output)

	return
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

func exportGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "export",
		Short: "export a landscape",
		Example: `	
export acls --dir my-dir
export alert-settings --dir my-dir
export connectors --dir my-dir --resource-name my-connector --cluster-name Cluster1
export processors --dir my-dir --resource-name my-processor
export quota --dir my-dir
export schemas --dir my-dir --resource-name my-schema-value --version 1
export topics --dir my-dir --resource-name my-topic
export policies --dir my-dir --resource-name my-policy`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.PersistentFlags().StringVar(&landscapeDir, "dir", ".", "Base directory to export to")
	cmd.PersistentFlags().BoolVar(&dependents, "dependents", false, "Extract dependencies, topics, acls, quotas, alerts")
	cmd.MarkPersistentFlagRequired("dir")
	cmd.AddCommand(exportAclsCommand())
	cmd.AddCommand(exportAlertsCommand())
	cmd.AddCommand(exportConnectorsCommand())
	cmd.AddCommand(exportProcessorsCommand())
	cmd.AddCommand(exportQuotasCommand())
	cmd.AddCommand(exportSchemasCommand())
	cmd.AddCommand(exportTopicsCommand())
	cmd.AddCommand(exportPoliciesCommand())

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
			if err := writeProcessors(cmd, id, cluster, namespace, name); err != nil {
				golog.Errorf("Error writing processors. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "", "The processor name to export")
	cmd.Flags().StringVar(&cluster, "cluster-name", "", "Select by cluster name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&namespace, "namespace", "", "Select by namespace, available only in KUBERNETES mode")
	cmd.Flags().StringVar(&id, "id", "", "ID of the processor to export")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Processor with the prefix in the name only")

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
			if err := writeTopics(cmd, name); err != nil {
				golog.Errorf("Error writing topics. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "", "The topic name to export")
	cmd.Flags().StringVar(&topicExclusions, "exclude", "", "Topics to exclude")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Topics with the prefix only")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func exportAlertsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "alert-settings",
		Short:            "export alert-settings",
		Example:          `export alert-settings --resource-name=my-alert`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)
			if err := writeAlertSetting(cmd); err != nil {
				golog.Errorf("Error writing alert-settings. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func exportAclsCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "export acls",
		Example:          `export acls`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			if err := writeACLs(cmd); err != nil {
				golog.Errorf("Error writing ACLS. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}

func exportQuotasCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:              "quotas",
		Short:            "export quotas",
		Example:          `export quoats`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			checkFileFlags(cmd)

			if err := writeQuotas(cmd); err != nil {
				golog.Errorf("Error writing quotas. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func exportConnectorsCommand() *cobra.Command {
	var name, cluster string

	cmd := &cobra.Command{
		Use:              "connectors",
		Short:            "export connectors",
		Example:          `export connectors --resource-name my-connector --cluster-name`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setExecutionMode()
			checkFileFlags(cmd)
			if err := writeConnectors(cmd, cluster, name); err != nil {
				golog.Errorf("Error writing connectors. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "", "The resource name to export")
	cmd.Flags().StringVar(&cluster, "cluster-name", "", "Select by cluster name, available only in CONNECT and KUBERNETES mode")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Connector with the prefix in the name only")
	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	return cmd
}

func exportPoliciesCommand() *cobra.Command {
	var name, ID string

	cmd := &cobra.Command{
		Use:              "policies",
		Short:            "export policies",
		Example:          `export policies --resource-name my-policy`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			setExecutionMode()
			checkFileFlags(cmd)
			if err := writePolicies(cmd, name, ID); err != nil {
				golog.Errorf("Error writing policies. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "", "The resource name to export")
	cmd.Flags().StringVar(&ID, "id", "", "The policy id to extract")
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

			if name != "" {
				if err := writeSchema(cmd, name, versionInt); err != nil {
					golog.Errorf("Error writing schema. [%s]", err.Error())
					return err
				}
				return nil
			}

			if err := writeSchemas(cmd); err != nil {
				golog.Errorf("Error writing schemas. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&name, "resource-name", "", "The schema to export. Both the key schema and value schema are exported")
	cmd.Flags().StringVar(&version, "version", "0", "The schema version to export.")
	cmd.Flags().StringVar(&prefix, "prefix", "", "Schemas with the prefix only")
	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)
	return cmd
}
