package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"

	"github.com/landoop/bite"

	"github.com/kataras/golog"
	"github.com/landoop/lenses-go"
	"github.com/spf13/cobra"
)

var importDir string

func init() {
	app.AddCommand(importGroupCommand())
}

func findFiles(dir string) []os.FileInfo {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		golog.Fatal(err)
	}
	return files
}

func loadTopics(cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading topics from [%s]", loadpath)
	files := findFiles(loadpath)
	topics, err := client.GetTopics()

	if err != nil {
		golog.Errorf("Error retrieving topics [%s]", err.Error())
		return err
	}

	for _, file := range files {
		var topic lenses.CreateTopicPayload
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &topic); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		found := false

		for _, lensesTopic := range topics {
			if lensesTopic.TopicName == topic.TopicName {
				found = true
				if err := client.UpdateTopic(topic.TopicName, []lenses.KV{topic.Configs}); err != nil {
					golog.Errorf("Error updating topic [%s]. [%s]", topic.TopicName, err.Error())
					return err
				}

				golog.Infof("Updated topic [%s]", topic.TopicName)
			}
		}

		if !found {
			if err := client.CreateTopic(topic.TopicName, topic.Replication, topic.Partitions, topic.Configs); err != nil {
				golog.Errorf("Error creating topic [%s]. [%s]", topic.TopicName, err.Error())
				return err
			}

			golog.Infof("Created topic [%s]", topic.TopicName)
		}
	}

	return nil
}

func loadAcls(cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading acls from [%s]", loadpath)
	files := findFiles(loadpath)

	lacls, err := client.GetACLs()

	if err != nil {
		return err
	}

	for _, file := range files {
		var acls []lenses.ACL
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &acls); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		found := true
		for _, l := range lacls {
			if acl.Host == l.Host &&
				acl.Operation == l.Operation &&
				acl.PermissionType == l.PermissionType &&
				acl.Principal == l.Principal &&
				acl.ResourceName == l.ResourceName &&
				acl.ResourceType == l.ResourceType {
				found = true
			}
		}

		if found {
			continue
		}

		for _, acl := range acls {
			if err := client.CreateOrUpdateACL(acl); err != nil {
				golog.Errorf("Error creating/updating acl from [%s] [%s]", loadpath, err.Error())
				return err
			}
		}

		golog.Infof("Created/updated ACLs from [%s]", loadpath)
	}
	return nil
}

func loadAlertSettings(cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading alert-settings from [%s]", loadpath)
	files := findFiles(loadpath)

	asc, err := client.GetAlertSettingConditions(2000)

	if err != nil {
		return err
	}

	for _, file := range files {

		var conds AlertSettingConditionPayloads
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &conds); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		alertID := conds.AlertID

		for _, condition := range conds.Conditions {
			found := false
			for _, v := range asc {
				if v == condition {
					found = true
				}
			}

			if found {
				continue
			}

			if err := client.CreateOrUpdateAlertSettingCondition(alertID, condition); err != nil {
				golog.Errorf("Error creating/updating alert setting from [%d] [%s] [%s]", alertID, loadpath, err.Error())
				return err
			}
			golog.Infof("Created/updated condition [%s] from [%s]", condition, loadpath)
		}
	}
	return nil
}

func loadQuotas(cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading quotas from [%s]", loadpath)
	files := findFiles(loadpath)

	lensesQuotas, err := client.GetQuotas()
	var lensesReq []lenses.CreateQuotaPayload

	if err != nil {
		return err
	}

	for _, lq := range lensesQuotas {
		lensesReq = append(lensesReq, lq.GetQuotaAsRequest())
	}

	for _, file := range files {
		var quotas []lenses.CreateQuotaPayload
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &quotas); err != nil {
			golog.Errorf("Error loading file [%s]", loadpath)
			return err
		}

		for _, quota := range quotas {

			found := false
			for _, lq := range lensesReq {
				if quota.ClientID == lq.ClientID &&
					quota.QuotaType == lq.QuotaType &&
					quota.User == lq.User &&
					quota.Config.ConsumerByteRate == quota.Config.ConsumerByteRate &&
					quota.Config.ProducerByteRate == quota.Config.ProducerByteRate &&
					quota.Config.RequestPercentage == quota.Config.RequestPercentage {
					found = true
				}
			}

			if found {
				continue
			}

			if quota.QuotaType == string(lenses.QuotaEntityClient) ||
				quota.QuotaType == string(lenses.QuotaEntityClients) ||
				quota.QuotaType == string(lenses.QuotaEntityClientsDefault) {
				if err := CreateQuotaForClients(cmd, quota); err != nil {
					golog.Errorf("Error creating/updating quota type [%s], client [%s], user [%s] from [%s]. [%s]",
						quota.QuotaType, quota.ClientID, quota.User, loadpath, err.Error())
					return err
				}

				golog.Infof("Created/updated quota type [%s], client [%s], user [%s] from [%s]",
					quota.QuotaType, quota.ClientID, quota.User, loadpath)
				continue

			}

			if err := CreateQuotaForUsers(cmd, quota); err != nil {
				golog.Errorf("Error creating/updating quota type [%s], client [%s], user [%s] from [%s]. [%s]",
					quota.QuotaType, quota.ClientID, quota.User, loadpath, err.Error())
				return err
			}

			golog.Infof("Created/updated quota type [%s], client [%s], user [%s] from [%s]",
				quota.QuotaType, quota.ClientID, quota.User, loadpath)
		}
	}
	return nil
}

func loadConnectors(cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading connectors from [%s]", loadpath)
	files := findFiles(loadpath)

	for _, file := range files {
		var connector lenses.CreateUpdateConnectorPayload
		if err := load(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &connector); err != nil {
			return err
		}

		connectors, err := client.GetConnectors(connector.ClusterName)

		if err != nil {
			return err
		}

		existsOrUpdated := false
		for _, name := range connectors {
			if name == connector.Name {
				c, err := client.GetConnector(connector.ClusterName, connector.Name)

				if err != nil {
					return err
				}

				if !reflect.DeepEqual(c.Config, connector.Config) {
					_, errU := client.UpdateConnector(connector.ClusterName, connector.Name, connector.Config)
					if errU != nil {
						golog.Errorf("Error updating connector from file [%s]. [%s]", loadpath, errU.Error())
						return errU
					}

					golog.Infof("Updated connector config for cluster [%s], connector [%s]", connector.ClusterName, connector.Name)
					break
				}

				existsOrUpdated = true
				break
			}
		}

		if existsOrUpdated {
			continue
		}
		_, errC := client.CreateConnector(connector.ClusterName, connector.Name, connector.Config)

		if errC != nil {
			golog.Errorf("Error creating connector from file [%s]. [%s]", loadpath, errC.Error())
			return err
		}

		golog.Infof("Created/updated connector from [%s]", loadpath)
	}

	return nil
}

func loadProcessors(cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading processors from [%s]", loadpath)
	files := findFiles(loadpath)

	processors, err := client.GetProcessors()

	if err != nil {
		golog.Errorf("Failed to retrieve processors. [%s]", err.Error())
	}

	for _, file := range files {

		var processor lenses.CreateProcessorPayload

		if err := load(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &processor); err != nil {
			return err
		}

		for _, p := range processors.Streams {
			if processor.Name == p.Name &&
				processor.ClusterName == p.ClusterName &&
				processor.Namespace == p.Namespace {

				if processor.Runners != p.Runners {
					//scale
					if err := client.UpdateProcessorRunners(p.ID, processor.Runners); err != nil {
						golog.Errorf("Error scaling processor [%s] from file [%s/%s]. [%s]", p.ID, loadpath, file.Name(), err.Error())
						return err
					}
					golog.Infof("Scaled processor [%s] from file [%s/%s] from [%d] to [%d]", p.ID, loadpath, file.Name(), p.Runners, processor.Runners)
				}
				golog.Warnf("Processor [%s] from file [%s/%s] already exists", p.ID, loadpath, file.Name())
				break
			}

			if err := client.CreateProcessor(
				processor.Name,
				processor.SQL,
				processor.Runners,
				processor.ClusterName,
				processor.Namespace,
				processor.Pipeline); err != nil {

				golog.Errorf("Error creating processor from file [%s/%s]. [%s]", loadpath, file.Name(), err.Error())
				return err
			}

			golog.Infof("Created processor from [%s/%s]", loadpath, file.Name())

		}
	}
	return nil
}

func loadSchemas(cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading schemas from [%s]", loadpath)
	files := findFiles(loadpath)

	for _, file := range files {
		var schema lenses.SchemaAsRequest
		if err := load(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &schema); err != nil {
			return err
		}

		_, err := client.RegisterSchema(schema.Name, schema.AvroSchema)

		if err != nil {
			golog.Errorf("Error creating schema from file [%s]. [%s]", loadpath, err.Error())
			return err
		}

		golog.Infof("Created schema from [%s]", loadpath)
	}

	return nil
}

func loadPolicies(cmd *cobra.Command, loadpath string) error {
	golog.Infof("Loading data policies from [%s]", loadpath)
	files := findFiles(loadpath)

	polices, err := client.GetPolicies()

	if err != nil {
		return err
	}

	for _, file := range files {

		var policy lenses.DataPolicyRequest
		if err := bite.LoadFile(cmd, fmt.Sprintf("%s/%s", loadpath, file.Name()), &policy); err != nil {
			return err
		}

		found := false

		for _, p := range polices {
			if p.Name == policy.Name {
				found = true

				payload := lenses.DataPolicyUpdateRequest{
					ID:          p.ID,
					Name:        p.Name,
					Category:    p.Category,
					ImpactType:  p.ImpactType,
					Obfuscation: p.Obfuscation,
					Fields:      p.Fields,
				}

				if err := client.UpdatePolicy(payload); err != nil {
					golog.Errorf("Error updating data policy [%s]. [%s]", p.Name, err.Error())
					return err
				}

				golog.Infof("Updated policy [%s]", p.Name)
			}
		}

		if !found {
			if err := client.CreatePolicy(policy); err != nil {
				golog.Errorf("Error creating data policy [%s]. [%s]", policy.Name, err.Error())
				return err
			}

			golog.Infof("Created data policy [%s]", policy.Name)
		}
	}

	return nil
}

func load(cmd *cobra.Command, path string, data interface{}) error {
	if err := bite.TryReadFile(path, data); err != nil {
		return err
	}
	return nil
}

func importProcessorsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "processors",
		Short:            "processors",
		Example:          `import processors --dir /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, sqlPath)
			if err := loadProcessors(cmd, path); err != nil {
				golog.Errorf("Failed to load processors. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func importConnectorsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "connectors",
		Short:            "connectors",
		Example:          `import processors --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, connectorsPath)
			if err := loadConnectors(cmd, path); err != nil {
				golog.Errorf("Failed to load connectors. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func importAclsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "acls",
		Short:            "acls",
		Example:          `import acls --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, sqlPath)
			if err := loadAcls(cmd, path); err != nil {
				golog.Errorf("Failed to load acls. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func importAlertSettingsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "alert-settings",
		Short:            "alert-settings",
		Example:          `import alert-settings --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, alertSettingsPath)
			if err := loadAlertSettings(cmd, path); err != nil {
				golog.Errorf("Failed to load alert-settings. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func importQuotasCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "quotas",
		Short:            "quotas",
		Example:          `import quotas --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, quotasPath)
			if err := loadQuotas(cmd, path); err != nil {
				golog.Errorf("Failed to load quotas. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func importTopicsCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "topics",
		Short:            "topics",
		Example:          `import topics --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, topicsPath)
			if err := loadTopics(cmd, path); err != nil {
				golog.Errorf("Failed to load topics. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func importSchemasCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "schemas",
		Short:            "schemas",
		Example:          `import schemas --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, schemasPath)
			if err := loadSchemas(cmd, path); err != nil {
				golog.Errorf("Failed to load schemas. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func importPoliciesCommand() *cobra.Command {
	var path string

	cmd := &cobra.Command{
		Use:              "policies",
		Short:            "policies",
		Example:          `import policies --landscape /my-landscape --ignore-errors`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			path = fmt.Sprintf("%s/%s", path, policiesPath)
			if err := loadPolicies(cmd, path); err != nil {
				golog.Errorf("Failed to load policies. [%s]", err.Error())
				return err
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&path, "dir", ".", "Base directory to import")

	bite.CanPrintJSON(cmd)
	bite.CanBeSilent(cmd)
	cmd.Flags().Set("silent", "true")
	return cmd
}

func importGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "import",
		Short: "import a landscape",
		Example: `
import acls --landscape my-acls-dir
import alert-settings --landscape my-acls-dir
import connectors --landscape my-acls-dir
import processors  --landscape my-acls-dir
import quota --landscape my-acls-dir
import schemas --landscape my-acls-dir
import topics --landscape my-acls-dir
import policies --landscape my-acls-dir`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.AddCommand(importAclsCommand())
	cmd.AddCommand(importAlertSettingsCommand())
	cmd.AddCommand(importConnectorsCommand())
	cmd.AddCommand(importProcessorsCommand())
	cmd.AddCommand(importQuotasCommand())
	cmd.AddCommand(importSchemasCommand())
	cmd.AddCommand(importTopicsCommand())
	cmd.AddCommand(importPoliciesCommand())

	return cmd
}
