package export

import (
	"fmt"
	"strings"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/v5/pkg"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/lensesio/lenses-go/v5/pkg/utils"

	"github.com/kataras/golog"
	"github.com/spf13/cobra"
)

const (
	connectorClassKey = "connector.class"
	sqlConnectorClass = "com.landoop.connect.SQL"
)

var (
	mode api.ExecutionMode

	// If set, export other resource types as well if the resource to be
	// exported references them.
	dependents bool

	landscapeDir string

	systemTopicExclusions = []string{
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
	topicExclusions string
	prefix          string
)

// NewExportGroupCommand creates the `export` command
func NewExportGroupCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export",
		Short: "export a landscape",
		Example: `
export acls --dir my-dir
export alert-settings --dir my-dir
export alert-channels
export connectors --dir my-dir --resource-name my-connector --cluster-name Cluster1
export processors --dir my-dir --resource-name my-processor
export quota --dir my-dir
export schemas --dir my-dir --resource-name my-schema-value --version 1
export topics --dir my-dir --resource-name my-topic
export policies --dir my-dir --resource-name my-policy
export connections --dir my-dir
export connections --dir my-dir --connection-id 1
export groups --dir groups
export topic-settings --dir topic-settings
export serviceaccounts --dir serviceaccounts`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.MarkPersistentFlagRequired("dir")
	cmd.AddCommand(NewExportAclsCommand())
	cmd.AddCommand(NewExportAlertsCommand())
	cmd.AddCommand(NewExportConnectorsCommand())
	cmd.AddCommand(NewExportProcessorsCommand())
	cmd.AddCommand(NewExportQuotasCommand())
	cmd.AddCommand(NewExportTopicsCommand())
	cmd.AddCommand(NewExportPoliciesCommand())
	cmd.AddCommand(NewExportConnectionsCommand())
	cmd.AddCommand(NewExportGroupsCommand())
	cmd.AddCommand(NewExportServiceAccountsCommand())
	cmd.AddCommand(NewExportAlertChannelsCommand())
	cmd.AddCommand(NewExportTopicSettingsCmd())
	cmd.AddCommand(NewExportAuditChannelsCommand())
	cmd.AddCommand(NewExportSchemasCmd())

	return cmd
}

func setExecutionMode(client *api.Client) error {
	execMode, err := getExecutionMode(client)
	if err != nil {
		return err
	}

	mode = execMode
	return nil
}

func getExecutionMode(client *api.Client) (api.ExecutionMode, error) {
	mode, err := client.GetExecutionMode()
	if err != nil {
		return mode, err
	}

	return mode, nil
}

// getAttachedTopics returns a slice of CreateTopicPayload for a topology node
// ID. If dependents is not set, it returns an empty slice.
func getAttachedTopics(client *api.Client, id string) ([]api.CreateTopicPayload, error) {
	if !dependents {
		return nil, nil
	}

	var topics []api.CreateTopicPayload

	extractedTopics, err := client.GetgetTopologyNodeGraph(id)
	if err != nil {
		return topics, err
	}

	for _, topicName := range extractedTopics {
		tree := append(topicName.Descendants, topicName.Parents...)

		for _, t := range tree {
			if strings.HasPrefix(t, "TOPIC-") {
				strippedTopicName := strings.Replace(t, "TOPIC-", "", len(t))
				topic, err := client.GetTopic(strippedTopicName)
				if err != nil {
					return topics, err
				}

				overrides := getTopicConfigOverrides(topic.Configs)
				topics = append(topics, topic.GetTopicAsRequest(overrides))
			}
		}
	}

	return topics, nil
}

func handleDependents(cmd *cobra.Command, client *api.Client, id string) error {
	// get topics
	topics, err := getAttachedTopics(client, id)
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
	settings, err := getConsumerAlertsByTopics(cmd, client, topicNames)
	if err != nil {
		return err
	}

	writeAlertSettingsAsRequest(cmd, settings)

	// get acls
	acls, err := client.GetACLs()
	if err != nil {
		return err
	}

	var topicAcls []api.ACL

	for _, acl := range acls {
		if acl.ResourceType == api.ACLResourceTopic {
			for _, topicName := range topicNames {
				if acl.ResourceName == topicName {
					topicAcls = append(topicAcls, acl)
				}
			}
		}
	}
	output := strings.ToUpper(bite.GetOutPutFlag(cmd))
	fileName := fmt.Sprintf("acls-%s.%s", "all", strings.ToLower(output))
	return utils.WriteFile(landscapeDir, pkg.AclsPath, fileName, output, topicAcls)
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
}
