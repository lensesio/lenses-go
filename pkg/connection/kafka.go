package connection

import (
	"github.com/lensesio/lenses-go/v5/pkg/api"
	cobra "github.com/spf13/cobra"
)

type kafkaClient interface {
	GetKafkaConnection(name string) (resp api.KafkaConnectionResponse, err error)
	UpdateKafkaConnection(name string, reqBody api.KafkaConnectionUpsertRequest) (resp api.AddConnectionResponse, err error)
	DeleteKafkaConnection(name string) (err error)
	TestKafkaConnection(reqBody api.KafkaConnectionTestRequest) (err error)
	ListKafkaConnections() (resp []api.ConnectionSummaryResponse, err error)
}

func NewKafkaGroupCommand(cl kafkaClient, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "kafka",
		Short:            "Manage Lenses Kafka connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	var testReq api.KafkaConnectionTestRequest
	var upsertReq api.KafkaConnectionUpsertRequest
	cmd.AddCommand(connCrudsToCobra("Kafka", up,
		connCrud{
			use:           "get",
			defaultArg:    "kafka",
			runWithArgRet: func(arg string) (interface{}, error) { return cl.GetKafkaConnection(arg) },
		}, connCrud{
			use:         "list",
			runNoArgRet: func() (interface{}, error) { return cl.ListKafkaConnections() },
		}, connCrud{
			use:        "test",
			defaultArg: "kafka",
			opts: FlagMapperOpts{
				Descriptions: kdescs,
				Hide:         []string{"Name", "MetricsCustomPortMappings"},
			},
			onto: &testReq,
			runWithargNoRet: func(arg string) error {
				testReq.Name = arg
				return cl.TestKafkaConnection(testReq)
			},
		}, connCrud{
			use:        "upsert",
			defaultArg: "kafka",
			opts: FlagMapperOpts{
				Descriptions: kdescs,
				Hide:         []string{"MetricsCustomPortMappings"},
			},
			onto: &upsertReq,
			runWithArgRet: func(arg string) (interface{}, error) {
				return cl.UpdateKafkaConnection(arg, upsertReq)
			},
		}, connCrud{
			use:             "delete",
			runWithargNoRet: cl.DeleteKafkaConnection,
		})...)

	return cmd
}

var kdescs = map[string]string{
	"KafkaBootstrapServers":     "Comma separated list of protocol://host:port to use for initial connection to Kafka.",
	"AdditionalProperties":      "Any other additional properties.",
	"Keytab":                    "Kerberos keytab file.",
	"MetricsCustomPortMappings": "DEPRECATED.",
	"MetricsCustomURLMappings":  "Mapping from node URL to metrics URL, allows overriding metrics target on a per-node basis.",
	"MetricsHTTPSuffix":         "HTTP URL suffix for Jolokia or AWS metrics.",
	"MetricsHTTPTimeout":        "HTTP Request timeout (ms) for Jolokia or AWS metrics.",
	"MetricsPassword":           "The password for metrics connections.",
	"MetricsPort":               "Default port number for metrics connection (JMX and JOLOKIA).",
	"MetricsSsl":                "Flag to enable SSL for metrics connections.",
	"MetricsType":               "Metrics type.",
	"MetricsUsername":           "The username for metrics connections.",
	"Protocol":                  "Kafka security protocol.",
	"SaslJaasConfig":            "JAAS Login module configuration for SASL.",
	"SaslMechanism":             "Mechanism to use when authenticated using SASL.",
	"SslKeyPassword":            "Key password for the keystore.",
	"SslKeystore":               "SSL keystore file.",
	"SslKeystorePassword":       "Password to the keystore.",
	"SslTruststore":             "SSL truststore file.",
	"SslTruststorePassword":     "Password to the truststore.",
	"Update":                    "Set to true if testing an update to an existing connection, false if testing a new connection.",
	"Tags":                      "Any tags to add to the connection's metadata.",
}
