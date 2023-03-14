package connection

import (
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	cobra "github.com/spf13/cobra"
)

func NewSchemaRegistryGroupCommand(up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "schema-registry",
		Aliases:          []string{"sr"},
		Short:            "Manage Lenses Schema Registry connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	var testReq api.SchemaRegistryConnectionTestRequest
	var upsertReq api.SchemaRegistryConnectionUpsertRequest
	cmd.AddCommand(connCrudsToCobra("Schema Registry", up,
		connCrud{
			use:           "get",
			defaultArg:    "schema-registry",
			runWithArgRet: func(arg string) (interface{}, error) { return config.Client.GetSchemaRegistryConnection(arg) },
		}, connCrud{
			use:         "list",
			runNoArgRet: func() (interface{}, error) { return config.Client.ListSchemaRegistryConnections() },
		}, connCrud{
			use:        "test",
			defaultArg: "schema-registry",
			opts: FlagMapperOpts{
				Descriptions: srdescs,
				Hide:         []string{"Name", "MetricsCustomPortMappings"},
			},
			onto: &testReq,
			runWithargNoRet: func(arg string) error {
				testReq.Name = arg
				return config.Client.TestSchemaRegistryConnection(testReq)
			},
		}, connCrud{
			use:        "upsert",
			defaultArg: "schema-registry",
			opts: FlagMapperOpts{
				Descriptions: srdescs,
				Hide:         []string{"MetricsCustomPortMappings"},
			},
			onto: &upsertReq,
			runWithArgRet: func(arg string) (interface{}, error) {
				return config.Client.UpdateSchemaRegistryConnection(arg, upsertReq)
			},
		}, connCrud{
			use:             "delete",
			runWithargNoRet: config.Client.DeleteSchemaRegistryConnection,
		})...)

	return cmd
}

var srdescs = map[string]string{
	"SchemaRegistryURLs":        "List of schema registry urls.",
	"AdditionalProperties":      "Any other additional properties.",
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
	"SslKeyPassword":            "Key password for the keystore.",
	"SslKeystore":               "SSL keystore.",
	"SslTruststore":             "SSL truststore.",
	"SslKeystorePassword":       "Password to the keystore.",
	"SslTruststorePassword":     "Password to the truststore.",
	"Username":                  "Username for HTTP Basic Authentication.",
	"Update":                    "Set to true if testing an update to an existing connection, false if testing a new connection.",
	"Tags":                      "Any tags to add to the connection's metadata.",
}
