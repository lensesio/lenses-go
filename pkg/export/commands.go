package export

import (
	"fmt"
	"os"
	"strings"

	"github.com/lensesio/bite"
	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	"github.com/lensesio/lenses-go/pkg/utils"

	"github.com/kataras/golog"
	"github.com/spf13/cobra"
	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
)

const (
	connectorClassKey = "connector.class"
	sqlConnectorClass = "com.landoop.connect.SQL"
)

var mode api.ExecutionMode
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

//NewExportGroupCommand creates the `export` command
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

func getAttachedTopics(client *api.Client, id string) ([]api.CreateTopicPayload, error) {
	var topics []api.CreateTopicPayload

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

func handleDependents(cmd *cobra.Command, client *api.Client, id string) error {

	//get topics
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
	settings, err := getAlertSettings(cmd, client, topicNames)

	if err != nil {
		return err
	}

	writeAlertSettingsAsRequest(cmd, settings)

	//get acls
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

	return
}
