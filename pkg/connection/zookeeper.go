package connection

import (
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	cobra "github.com/spf13/cobra"
)

func NewZookeeperGroupCommand(up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "zookeeper",
		Aliases:          []string{"zk"},
		Short:            "Manage Lenses Zookeeper connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	var testReq api.ZookeeperConnectionTestRequest
	var upsertReq api.ZookeeperConnectionUpsertRequest
	cmd.AddCommand(connCrudsToCobra("Zookeeper", up,
		connCrud{
			use:           "get",
			defaultArg:    "zookeeper",
			runWithArgRet: func(arg string) (interface{}, error) { return config.Client.GetZookeeperConnection(arg) },
		}, connCrud{
			use:         "list",
			runNoArgRet: func() (interface{}, error) { return config.Client.ListZookeeperConnections() },
		}, connCrud{
			use:        "test",
			defaultArg: "zookeeper",
			opts: FlagMapperOpts{
				Descriptions: zkdescs,
				Hide:         []string{"Name", "MetricsCustomPortMappings"},
			},
			onto: &testReq,
			runWithargNoRet: func(arg string) error {
				testReq.Name = arg
				return config.Client.TestZookeeperConnection(testReq)
			},
		}, connCrud{
			use:        "upsert",
			defaultArg: "zookeeper",
			opts: FlagMapperOpts{
				Descriptions: zkdescs,
				Hide:         []string{"MetricsCustomPortMappings"},
			},
			onto: &upsertReq,
			runWithArgRet: func(arg string) (interface{}, error) {
				return config.Client.UpdateZookeeperConnection(arg, upsertReq)
			},
		}, connCrud{
			use:             "delete",
			runWithargNoRet: config.Client.DeleteZookeeperConnection,
		})...)

	return cmd
}

var zkdescs = map[string]string{
	"ZookeeperConnectionTimeout": "Zookeeper connection timeout.",
	"ZookeeperSessionTimeout":    "Zookeeper connection session timeout.",
	"ZookeeperURLs":              "List of zookeeper urls.",
	"MetricsCustomPortMappings":  "DEPRECATED.",
	"MetricsCustomURLMappings":   "Mapping from node URL to metrics URL, allows overriding metrics target on a per-node basis.",
	"MetricsHTTPSuffix":          "HTTP URL suffix for Jolokia metrics.",
	"MetricsHTTPTimeout":         "HTTP Request timeout (ms) for Jolokia metrics.",
	"MetricsPassword":            "The password for metrics connections.",
	"MetricsPort":                "Default port number for metrics connection (JMX and JOLOKIA).",
	"MetricsSsl":                 "Flag to enable SSL for metrics connections.",
	"MetricsType":                "Metrics type.",
	"MetricsUsername":            "The username for metrics connections.",
	"ZookeeperChrootPath":        "Zookeeper /znode path.",
	"Update":                     "Set to true if testing an update to an existing connection, false if testing a new connection.",
	"Tags":                       "Any tags to add to the connection's metadata.",
}
