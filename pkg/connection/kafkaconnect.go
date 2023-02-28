package connection

import (
	"github.com/lensesio/lenses-go/pkg/api"
	config "github.com/lensesio/lenses-go/pkg/configs"
	cobra "github.com/spf13/cobra"
)

func NewKafkaConnectGroupCommand(up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "kafka-connect",
		Aliases:          []string{"kc"},
		Short:            "Manage Lenses Kafka Connect connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	var testReq api.KafkaConnectConnectionTestRequest
	var upsertReq api.KafkaConnectConnectionUpsertRequest
	cmd.AddCommand(connCrudsToCobra("Kafka Connect", up,
		connCrud{
			use:           "get",
			runWithArgRet: func(arg string) (interface{}, error) { return config.Client.GetKafkaConnectConnection(arg) },
		}, connCrud{
			use:         "list",
			runNoArgRet: func() (interface{}, error) { return config.Client.ListKafkaConnectConnections() },
		}, connCrud{
			use: "test",
			opts: FlagMapperOpts{
				Descriptions: kcdescs,
				Hide:         []string{"Name", "MetricsCustomPortMappings"},
			},
			onto: &testReq,
			runWithargNoRet: func(arg string) error {
				testReq.Name = arg
				return config.Client.TestKafkaConnectConnection(testReq)
			},
		}, connCrud{
			use: "upsert",
			opts: FlagMapperOpts{
				Descriptions: kcdescs,
				Hide:         []string{"MetricsCustomPortMappings"},
			},
			onto: &upsertReq,
			runWithArgRet: func(arg string) (interface{}, error) {
				return config.Client.UpdateKafkaConnectConnection(arg, upsertReq)
			},
		}, connCrud{
			use:             "delete",
			runWithargNoRet: config.Client.DeleteKafkaConnectConnection,
		})...)

	return cmd
}

var kcdescs = map[string]string{
	"Workers":                   "List of Kafka Connect worker URLs.",
	"Aes256Key":                 "AES256 Key used to encrypt secret properties when deploying Connectors to this ConnectCluster.",
	"MetricsCustomPortMappings": "DEPRECATED.",
	"MetricsCustomURLMappings":  "Mapping from node URL to metrics URL, allows overriding metrics target on a per-node basis.",
	"MetricsHTTPSuffix":         "HTTP URL suffix for Jolokia metrics.",
	"MetricsHTTPTimeout":        "HTTP Request timeout (ms) for Jolokia metrics.",
	"MetricsPassword":           "The password for metrics connections.",
	"MetricsPort":               "Default port number for metrics connection (JMX and JOLOKIA).",
	"MetricsSsl":                "Flag to enable SSL for metrics connections.",
	"MetricsType":               "Metrics type.",
	"MetricsUsername":           "The username for metrics connections.",
	"Password":                  "Password for HTTP Basic Authentication.",
	"SslAlgorithm":              "Name of the ssl algorithm. If empty default one will be used (X509).",
	"SslKeyPassword":            "Key password for the keystore.",
	"SslKeystore":               "SSL keystore file.",
	"SslKeystorePassword":       "Password to the keystore.",
	"SslTruststore":             "SSL truststore file.",
	"SslTruststorePassword":     "Password to the truststore.",
	"Username":                  "Username for HTTP Basic Authentication.",
	"Update":                    "Set to true if testing an update to an existing connection, false if testing a new connection.",
	"Tags":                      "Any tags to add to the connection's metadata.",
}
