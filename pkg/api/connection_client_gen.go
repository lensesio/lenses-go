// This code is automatically generated based on the open api spec of:
// Lenses API, version: v5.0.0-12-g2721a0e54.
package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

type UpsertConnectionAPIRequest struct {
	ConfigurationObject any      `json:"configurationObject,omitempty"` // Optional. The configuration of the connection. The schema of this object is defined by the [template configuration](#operation/listConnectionTemplates).
	Tags                []string `json:"tags"`                          // Optional.
	TemplateName        *string  `json:"templateName,omitempty"`        // Optional. The [template](#operation/listConnectionTemplates) of the connection.
}

type TestConnectionAPIRequest struct {
	Name                string `json:"name"`                          // Required.
	TemplateName        string `json:"templateName"`                  // Required. The [template](#operation/listConnectionTemplates) of the connection.
	ConfigurationObject any    `json:"configurationObject,omitempty"` // Optional. The configuration of the connection. The schema of this object is defined by the [template configuration](#operation/listConnectionTemplates).
	Update              *bool  `json:"update,omitempty"`              // Optional. *true* if testing an update to an existing connection, *false* if testing a new connection.
}

type Json map[string]interface{}

type ConnectionPropertyJsonResponse struct {
	Key     string      `json:"key"`     // Required.
	Mounted bool        `json:"mounted"` // Required.
	Type    KeyDataType `json:"type"`    // Required. The data type of the property.
	Value   any         `json:"value"`   // Required.
}

type ConnectionJsonResponse struct {
	BuiltIn             bool                             `json:"builtIn"`                 // Required.
	ConfigurationObject map[string]interface{}           `json:"configurationObject"`     // Required. The configuration of the connection. The schema of this object is defined by the [template configuration](#operation/listConnectionTemplates).
	CreatedAt           int                              `json:"createdAt"`               // Required.
	CreatedBy           string                           `json:"createdBy"`               // Required.
	Deletable           bool                             `json:"deletable"`               // Required.
	ModifiedAt          int                              `json:"modifiedAt"`              // Required.
	ModifiedBy          string                           `json:"modifiedBy"`              // Required.
	Name                string                           `json:"name"`                    // Required. Name of the connection.
	TemplateName        string                           `json:"templateName"`            // Required. The [template](#operation/listConnectionTemplates) of the connection.
	TemplateVersion     int                              `json:"templateVersion"`         // Required.
	Configuration       []ConnectionPropertyJsonResponse `json:"configuration,omitempty"` // Deprecated. Optional.
	Tags                []string                         `json:"tags,omitempty"`          // Optional.
}

type ConnectionTemplateMetadata2 struct {
	Author      string  `json:"author"`                // Required.
	Description *string `json:"description,omitempty"` // Optional.
}

type JsonObject map[string]interface{}

type KeyDataType string

type ConnectionPropertyType struct {
	DisplayName string      `json:"displayName"` // Required.
	Name        KeyDataType `json:"name"`        // Required. The data type of the property.
}

type ConnectionProperty struct {
	DisplayName string                 `json:"displayName"`           // Required.
	Key         string                 `json:"key"`                   // Required. The key of the property.
	Mounted     bool                   `json:"mounted"`               // Required. Denotes a file property which expects an [uploaded file reference](#operation/uploadFile) as its value.
	Provided    bool                   `json:"provided"`              // Required.
	Required    bool                   `json:"required"`              // Required.
	Type        ConnectionPropertyType `json:"type"`                  // Required.
	Description *string                `json:"description,omitempty"` // Optional.
	EnumValues  []string               `json:"enumValues,omitempty"`  // Optional. For enum properties, the set of valid values.
	Placeholder *string                `json:"placeholder,omitempty"` // Optional.
}

type TemplateCategory string

const (
	TemplateCategoryConnection  TemplateCategory = "Connection"
	TemplateCategoryApplication TemplateCategory = "Application"
	TemplateCategoryChannel     TemplateCategory = "Channel"
	TemplateCategoryDeployment  TemplateCategory = "Deployment"
)

type TemplateType string

const (
	TemplateTypeNotApplicable TemplateType = "Not Applicable"
	TemplateTypeSQLRunner     TemplateType = "SQLRunner"
	TemplateTypeKubernetes    TemplateType = "Kubernetes"
	TemplateTypeInProc        TemplateType = "InProc"
	TemplateTypeConnect       TemplateType = "Connect"
	TemplateTypeAlertChannel  TemplateType = "Alert Channel"
	TemplateTypeAuditChannel  TemplateType = "Audit Channel"
)

type ConnectionTemplateResponse struct {
	BuiltIn         bool                        `json:"builtIn"`                 // Required.
	Category        TemplateCategory            `json:"category"`                // Required.
	Creatable       bool                        `json:"creatable"`               // Required.
	Enabled         bool                        `json:"enabled"`                 // Required.
	JsonSchema      JsonObject                  `json:"jsonSchema"`              // Required. JSON Schema representation of the configuration properties.
	Metadata        ConnectionTemplateMetadata2 `json:"metadata"`                // Required.
	Name            string                      `json:"name"`                    // Required. The name of the template.
	TemplateVersion int                         `json:"templateVersion"`         // Required.
	Type            TemplateType                `json:"type"`                    // Required.
	Version         string                      `json:"version"`                 // Required.
	Configuration   []ConnectionProperty        `json:"configuration,omitempty"` // Optional. Array of objects describing the schema of each configuration property for connections of this type.
}

type ConnectionSummaryResponse struct {
	Deletable       bool     `json:"deletable"`       // Required. Can the connection be deleted.
	Name            string   `json:"name"`            // Required. Name of the connection.
	TemplateName    string   `json:"templateName"`    // Required. The [template](#operation/listConnectionTemplates) of the connection.
	TemplateVersion int      `json:"templateVersion"` // Required.
	Tags            []string `json:"tags,omitempty"`  // Optional.
}

type AddConnectionResponse struct {
	Name string `json:"name"` // Required. Name of the connection.
}

type SchemaRegistryConnectionConfiguration struct {
	SchemaRegistryURLs        []string          `json:"schemaRegistryUrls"`                  // Required. List of schema registry urls.
	AdditionalProperties      map[string]string `json:"additionalProperties,omitempty"`      // Optional.
	MetricsCustomPortMappings map[string]string `json:"metricsCustomPortMappings,omitempty"` // Optional. DEPRECATED.
	MetricsCustomURLMappings  map[string]string `json:"metricsCustomUrlMappings,omitempty"`  // Optional. Mapping from node URL to metrics URL, allows overriding metrics target on a per-node basis.
	MetricsHTTPSuffix         *string           `json:"metricsHttpSuffix,omitempty"`         // Optional. HTTP URL suffix for Jolokia metrics.
	MetricsHTTPTimeout        *int              `json:"metricsHttpTimeout,omitempty"`        // Optional. HTTP Request timeout (ms) for Jolokia metrics.
	MetricsPassword           *string           `json:"metricsPassword,omitempty"`           // Optional. The password for metrics connections.
	MetricsPort               *int              `json:"metricsPort,omitempty"`               // Optional. Default port number for metrics connection (JMX and JOLOKIA).
	MetricsSsl                *bool             `json:"metricsSsl,omitempty"`                // Optional. Flag to enable SSL for metrics connections.
	MetricsType               *string           `json:"metricsType,omitempty"`               // Optional. Metrics type.
	MetricsUsername           *string           `json:"metricsUsername,omitempty"`           // Optional. The username for metrics connections.
	Password                  *string           `json:"password,omitempty"`                  // Optional. Password for HTTP Basic Authentication.
	SslKeyPassword            *string           `json:"sslKeyPassword,omitempty"`            // Optional. Key password for the keystore.
	SslKeystore               *struct {
		FileId uuid.UUID `json:"fileId"` // Required.
	} `json:"sslKeystore,omitempty"` // Optional. SSL keystore file.
	SslKeystorePassword *string `json:"sslKeystorePassword,omitempty"` // Optional. Password to the keystore.
	SslTruststore       *struct {
		FileId uuid.UUID `json:"fileId"` // Required.
	} `json:"sslTruststore,omitempty"` // Optional. SSL truststore file.
	SslTruststorePassword *string `json:"sslTruststorePassword,omitempty"` // Optional. Password to the truststore.
	Username              *string `json:"username,omitempty"`              // Optional. Username for HTTP Basic Authentication.
}

type SchemaRegistryConnectionAddRequest struct {
	ConfigurationObject SchemaRegistryConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                                `json:"name"`                // Required. Name of the connection.
	Tags                []string                              `json:"tags"`                // Optional.
}

type SchemaRegistryConnectionTestRequest struct {
	ConfigurationObject SchemaRegistryConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                                `json:"name"`                // Required. Name of the connection.
	Update              *bool                                 `json:"update,omitempty"`    // Optional. *true* if testing an update to an existing connection, *false* if testing a new connection.
}

type SchemaRegistryConnectionResponse struct {
	BuiltIn             bool                                  `json:"builtIn"`             // Required.
	ConfigurationObject SchemaRegistryConnectionConfiguration `json:"configurationObject"` // Required.
	CreatedAt           int                                   `json:"createdAt"`           // Required.
	CreatedBy           string                                `json:"createdBy"`           // Required.
	Deletable           bool                                  `json:"deletable"`           // Required.
	ModifiedAt          int                                   `json:"modifiedAt"`          // Required.
	ModifiedBy          string                                `json:"modifiedBy"`          // Required.
	Name                string                                `json:"name"`                // Required. Name of the connection.
	TemplateName        string                                `json:"templateName"`        // Required. The [template](#operation/listConnectionTemplates) of the connection.
	TemplateVersion     int                                   `json:"templateVersion"`     // Required.
	Tags                []string                              `json:"tags,omitempty"`      // Optional.
}

type ZookeeperConnectionConfiguration struct {
	ZookeeperConnectionTimeout int               `json:"zookeeperConnectionTimeout"`          // Required. Zookeeper connection timeout.
	ZookeeperSessionTimeout    int               `json:"zookeeperSessionTimeout"`             // Required. Zookeeper connection session timeout.
	ZookeeperURLs              []string          `json:"zookeeperUrls"`                       // Required. List of zookeeper urls.
	MetricsCustomPortMappings  map[string]string `json:"metricsCustomPortMappings,omitempty"` // Optional. DEPRECATED.
	MetricsCustomURLMappings   map[string]string `json:"metricsCustomUrlMappings,omitempty"`  // Optional. Mapping from node URL to metrics URL, allows overriding metrics target on a per-node basis.
	MetricsHTTPSuffix          *string           `json:"metricsHttpSuffix,omitempty"`         // Optional. HTTP URL suffix for Jolokia metrics.
	MetricsHTTPTimeout         *int              `json:"metricsHttpTimeout,omitempty"`        // Optional. HTTP Request timeout (ms) for Jolokia metrics.
	MetricsPassword            *string           `json:"metricsPassword,omitempty"`           // Optional. The password for metrics connections.
	MetricsPort                *int              `json:"metricsPort,omitempty"`               // Optional. Default port number for metrics connection (JMX and JOLOKIA).
	MetricsSsl                 *bool             `json:"metricsSsl,omitempty"`                // Optional. Flag to enable SSL for metrics connections.
	MetricsType                *string           `json:"metricsType,omitempty"`               // Optional. Metrics type.
	MetricsUsername            *string           `json:"metricsUsername,omitempty"`           // Optional. The username for metrics connections.
	ZookeeperChrootPath        *string           `json:"zookeeperChrootPath,omitempty"`       // Optional. Zookeeper /znode path.
}

type ZookeeperConnectionAddRequest struct {
	ConfigurationObject ZookeeperConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                           `json:"name"`                // Required. Name of the connection.
	Tags                []string                         `json:"tags"`                // Optional.
}

type ZookeeperConnectionResponse struct {
	BuiltIn             bool                             `json:"builtIn"`             // Required.
	ConfigurationObject ZookeeperConnectionConfiguration `json:"configurationObject"` // Required.
	CreatedAt           int                              `json:"createdAt"`           // Required.
	CreatedBy           string                           `json:"createdBy"`           // Required.
	Deletable           bool                             `json:"deletable"`           // Required.
	ModifiedAt          int                              `json:"modifiedAt"`          // Required.
	ModifiedBy          string                           `json:"modifiedBy"`          // Required.
	Name                string                           `json:"name"`                // Required. Name of the connection.
	TemplateName        string                           `json:"templateName"`        // Required. The [template](#operation/listConnectionTemplates) of the connection.
	TemplateVersion     int                              `json:"templateVersion"`     // Required.
	Tags                []string                         `json:"tags,omitempty"`      // Optional.
}

type SchemaRegistryConnectionUpsertRequest struct {
	ConfigurationObject SchemaRegistryConnectionConfiguration `json:"configurationObject"` // Required.
	Tags                []string                              `json:"tags"`
}

type ZookeeperConnectionTestRequest struct {
	ConfigurationObject ZookeeperConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                           `json:"name"`                // Required. Name of the connection.
	Update              *bool                            `json:"update,omitempty"`    // Optional. *true* if testing an update to an existing connection, *false* if testing a new connection.
}

type ZookeeperConnectionUpsertRequest struct {
	ConfigurationObject ZookeeperConnectionConfiguration `json:"configurationObject"` // Required.
	Tags                []string                         `json:"tags"`
}

type KafkaConnectConnectionConfiguration struct {
	Workers                   []string          `json:"workers"`                             // Required. List of Kafka Connect worker URLs.
	Aes256Key                 *string           `json:"aes256Key,omitempty"`                 // Optional. AES256 Key used to encrypt secret properties when deploying Connectors to this ConnectCluster.
	MetricsCustomPortMappings map[string]string `json:"metricsCustomPortMappings,omitempty"` // Optional. DEPRECATED.
	MetricsCustomURLMappings  map[string]string `json:"metricsCustomUrlMappings,omitempty"`  // Optional. Mapping from node URL to metrics URL, allows overriding metrics target on a per-node basis.
	MetricsHTTPSuffix         *string           `json:"metricsHttpSuffix,omitempty"`         // Optional. HTTP URL suffix for Jolokia metrics.
	MetricsHTTPTimeout        *int              `json:"metricsHttpTimeout,omitempty"`        // Optional. HTTP Request timeout (ms) for Jolokia metrics.
	MetricsPassword           *string           `json:"metricsPassword,omitempty"`           // Optional. The password for metrics connections.
	MetricsPort               *int              `json:"metricsPort,omitempty"`               // Optional. Default port number for metrics connection (JMX and JOLOKIA).
	MetricsSsl                *bool             `json:"metricsSsl,omitempty"`                // Optional. Flag to enable SSL for metrics connections.
	MetricsType               *string           `json:"metricsType,omitempty"`               // Optional. Metrics type.
	MetricsUsername           *string           `json:"metricsUsername,omitempty"`           // Optional. The username for metrics connections.
	Password                  *string           `json:"password,omitempty"`                  // Optional. Password for HTTP Basic Authentication.
	SslAlgorithm              *string           `json:"sslAlgorithm,omitempty"`              // Optional. Name of the ssl algorithm. If empty default one will be used (X509).
	SslKeyPassword            *string           `json:"sslKeyPassword,omitempty"`            // Optional. Key password for the keystore.
	SslKeystore               *struct {
		FileId uuid.UUID `json:"fileId"` // Required.
	} `json:"sslKeystore,omitempty"` // Optional. SSL keystore file.
	SslKeystorePassword *string `json:"sslKeystorePassword,omitempty"` // Optional. Password to the keystore.
	SslTruststore       *struct {
		FileId uuid.UUID `json:"fileId"` // Required.
	} `json:"sslTruststore,omitempty"` // Optional. SSL truststore file.
	SslTruststorePassword *string `json:"sslTruststorePassword,omitempty"` // Optional. Password to the truststore.
	Username              *string `json:"username,omitempty"`              // Optional. Username for HTTP Basic Authentication.
}

type KafkaConnectConnectionAddRequest struct {
	ConfigurationObject KafkaConnectConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                              `json:"name"`                // Required. Name of the connection.
	Tags                []string                            `json:"tags"`                // Optional.
}

type KafkaConnectConnectionResponse struct {
	BuiltIn             bool                                `json:"builtIn"`             // Required.
	ConfigurationObject KafkaConnectConnectionConfiguration `json:"configurationObject"` // Required.
	CreatedAt           int                                 `json:"createdAt"`           // Required.
	CreatedBy           string                              `json:"createdBy"`           // Required.
	Deletable           bool                                `json:"deletable"`           // Required.
	ModifiedAt          int                                 `json:"modifiedAt"`          // Required.
	ModifiedBy          string                              `json:"modifiedBy"`          // Required.
	Name                string                              `json:"name"`                // Required. Name of the connection.
	TemplateName        string                              `json:"templateName"`        // Required. The [template](#operation/listConnectionTemplates) of the connection.
	TemplateVersion     int                                 `json:"templateVersion"`     // Required.
	Tags                []string                            `json:"tags,omitempty"`      // Optional.
}

type KafkaConnectConnectionTestRequest struct {
	ConfigurationObject KafkaConnectConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                              `json:"name"`                // Required. Name of the connection.
	Update              *bool                               `json:"update,omitempty"`    // Optional. *true* if testing an update to an existing connection, *false* if testing a new connection.
}

type KafkaConnectConnectionUpsertRequest struct {
	ConfigurationObject KafkaConnectConnectionConfiguration `json:"configurationObject"` // Required.
	Tags                []string                            `json:"tags"`
}

type KafkaConnectionConfiguration struct {
	KafkaBootstrapServers []string          `json:"kafkaBootstrapServers"`          // Required. Comma separated list of protocol://host:port to use for initial connection to Kafka.
	AdditionalProperties  map[string]string `json:"additionalProperties,omitempty"` // Optional.
	Keytab                *struct {
		FileId uuid.UUID `json:"fileId"` // Required.
	} `json:"keytab,omitempty"` // Optional. Kerberos keytab file.
	MetricsCustomPortMappings map[string]string `json:"metricsCustomPortMappings,omitempty"` // Optional. DEPRECATED.
	MetricsCustomURLMappings  map[string]string `json:"metricsCustomUrlMappings,omitempty"`  // Optional. Mapping from node URL to metrics URL, allows overriding metrics target on a per-node basis.
	MetricsHTTPSuffix         *string           `json:"metricsHttpSuffix,omitempty"`         // Optional. HTTP URL suffix for Jolokia or AWS metrics.
	MetricsHTTPTimeout        *int              `json:"metricsHttpTimeout,omitempty"`        // Optional. HTTP Request timeout (ms) for Jolokia or AWS metrics.
	MetricsPassword           *string           `json:"metricsPassword,omitempty"`           // Optional. The password for metrics connections.
	MetricsPort               *int              `json:"metricsPort,omitempty"`               // Optional. Default port number for metrics connection (JMX and JOLOKIA).
	MetricsSsl                *bool             `json:"metricsSsl,omitempty"`                // Optional. Flag to enable SSL for metrics connections.
	MetricsType               *string           `json:"metricsType,omitempty"`               // Optional. Metrics type.
	MetricsUsername           *string           `json:"metricsUsername,omitempty"`           // Optional. The username for metrics connections.
	Protocol                  *string           `json:"protocol,omitempty"`                  // Optional. Kafka security protocol.
	SaslJaasConfig            *string           `json:"saslJaasConfig,omitempty"`            // Optional. JAAS Login module configuration for SASL.
	SaslMechanism             *string           `json:"saslMechanism,omitempty"`             // Optional. Mechanism to use when authenticated using SASL.
	SslKeyPassword            *string           `json:"sslKeyPassword,omitempty"`            // Optional. Key password for the keystore.
	SslKeystore               *struct {
		FileId uuid.UUID `json:"fileId"` // Required.
	} `json:"sslKeystore,omitempty"` // Optional. SSL keystore file.
	SslKeystorePassword *string `json:"sslKeystorePassword,omitempty"` // Optional. Password to the keystore.
	SslTruststore       *struct {
		FileId uuid.UUID `json:"fileId"` // Required.
	} `json:"sslTruststore,omitempty"` // Optional. SSL truststore file.
	SslTruststorePassword *string `json:"sslTruststorePassword,omitempty"` // Optional. Password to the truststore.
}

type KafkaConnectionAddRequest struct {
	ConfigurationObject KafkaConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                       `json:"name"`                // Required. Name of the connection.
	Tags                []string                     `json:"tags"`
}

type KafkaConnectionResponse struct {
	BuiltIn             bool                         `json:"builtIn"`             // Required.
	ConfigurationObject KafkaConnectionConfiguration `json:"configurationObject"` // Required.
	CreatedAt           int                          `json:"createdAt"`           // Required.
	CreatedBy           string                       `json:"createdBy"`           // Required.
	Deletable           bool                         `json:"deletable"`           // Required.
	ModifiedAt          int                          `json:"modifiedAt"`          // Required.
	ModifiedBy          string                       `json:"modifiedBy"`          // Required.
	Name                string                       `json:"name"`                // Required. Name of the connection.
	TemplateName        string                       `json:"templateName"`        // Required. The [template](#operation/listConnectionTemplates) of the connection.
	TemplateVersion     int                          `json:"templateVersion"`     // Required.
	Tags                []string                     `json:"tags,omitempty"`      // Optional.
}

type KafkaConnectionTestRequest struct {
	ConfigurationObject KafkaConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                       `json:"name"`                // Required. Name of the connection.
	Update              *bool                        `json:"update,omitempty"`    // Optional. *true* if testing an update to an existing connection, *false* if testing a new connection.
}

type KafkaConnectionUpsertRequest struct {
	ConfigurationObject KafkaConnectionConfiguration `json:"configurationObject"` // Required.
	Tags                []string                     `json:"tags"`
}

type KerberosConnectionConfiguration struct {
	KerberosKrb5 struct {
		FileId uuid.UUID `json:"fileId"` // Required.
	} `json:"kerberosKrb5"` // Required. Kerberos krb5.conf file.
}

type KerberosConnectionAddRequest struct {
	ConfigurationObject KerberosConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                          `json:"name"`                // Required. Name of the connection.
	Tags                []string                        `json:"tags"`
}

type KerberosConnectionResponse struct {
	BuiltIn             bool                            `json:"builtIn"`             // Required.
	ConfigurationObject KerberosConnectionConfiguration `json:"configurationObject"` // Required.
	CreatedAt           int                             `json:"createdAt"`           // Required.
	CreatedBy           string                          `json:"createdBy"`           // Required.
	Deletable           bool                            `json:"deletable"`           // Required.
	ModifiedAt          int                             `json:"modifiedAt"`          // Required.
	ModifiedBy          string                          `json:"modifiedBy"`          // Required.
	Name                string                          `json:"name"`                // Required. Name of the connection.
	TemplateName        string                          `json:"templateName"`        // Required. The [template](#operation/listConnectionTemplates) of the connection.
	TemplateVersion     int                             `json:"templateVersion"`     // Required.
	Tags                []string                        `json:"tags,omitempty"`      // Optional.
}

type KerberosConnectionTestRequest struct {
	ConfigurationObject KerberosConnectionConfiguration `json:"configurationObject"` // Required.
	Name                string                          `json:"name"`                // Required. Name of the connection.
	Update              *bool                           `json:"update,omitempty"`    // Optional. *true* if testing an update to an existing connection, *false* if testing a new connection.
}

type KerberosConnectionUpsertRequest struct {
	ConfigurationObject KerberosConnectionConfiguration `json:"configurationObject"` // Required.
	Tags                []string                        `json:"tags"`
}

// Returns the list of available connections.
// Tags: KafkaConnectConnections.
func (c *Client) ListKafkaConnectConnections() (resp []ConnectionSummaryResponse, err error) {
	err = c.do(
		http.MethodGet,
		"/api/v1/connection/connection-templates/KafkaConnect/connections",
		nil,   // request
		&resp, // response
	)
	return
}

// Adds a new connection.
// Tags: KafkaConnectConnections.
func (c *Client) CreateKafkaConnectConnection(reqBody KafkaConnectConnectionAddRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/KafkaConnect/connections",
		reqBody, // request
		&resp,   // response
	)
	return
}

// Validates the connection.
// Tags: KafkaConnectConnections.
func (c *Client) TestKafkaConnectConnection(reqBody KafkaConnectConnectionTestRequest) (err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/KafkaConnect/connections/test",
		reqBody, // request
		nil,     // response
	)
	return
}

// Returns the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: KafkaConnectConnections.
func (c *Client) GetKafkaConnectConnection(name string) (resp KafkaConnectConnectionResponse, err error) {
	err = c.do(
		http.MethodGet,
		fmt.Sprintf("/api/v1/connection/connection-templates/KafkaConnect/connections/%s", name),
		nil,   // request
		&resp, // response
	)
	return
}

// Updates the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: KafkaConnectConnections.
func (c *Client) UpdateKafkaConnectConnection(name string, reqBody KafkaConnectConnectionUpsertRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPut,
		fmt.Sprintf("/api/v1/connection/connection-templates/KafkaConnect/connections/%s", name),
		reqBody, // request
		&resp,   // response
	)
	return
}

// Deletes the connection.
// Parameters:
// - name: The name of the connection.
// Tags: KafkaConnectConnections.
func (c *Client) DeleteKafkaConnectConnection(name string) (err error) {
	err = c.do(
		http.MethodDelete,
		fmt.Sprintf("/api/v1/connection/connection-templates/KafkaConnect/connections/%s", name),
		nil, // request
		nil, // response
	)
	return
}

// Returns the list of available connections.
// Tags: KafkaConnections.
func (c *Client) ListKafkaConnections() (resp []ConnectionSummaryResponse, err error) {
	err = c.do(
		http.MethodGet,
		"/api/v1/connection/connection-templates/Kafka/connections",
		nil,   // request
		&resp, // response
	)
	return
}

// Adds a new connection.
// Tags: KafkaConnections.
func (c *Client) CreateKafkaConnection(reqBody KafkaConnectionAddRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/Kafka/connections",
		reqBody, // request
		&resp,   // response
	)
	return
}

// Validates the connection.
// Tags: KafkaConnections.
func (c *Client) TestKafkaConnection(reqBody KafkaConnectionTestRequest) (err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/Kafka/connections/test",
		reqBody, // request
		nil,     // response
	)
	return
}

// Returns the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: KafkaConnections.
func (c *Client) GetKafkaConnection(name string) (resp KafkaConnectionResponse, err error) {
	err = c.do(
		http.MethodGet,
		fmt.Sprintf("/api/v1/connection/connection-templates/Kafka/connections/%s", name),
		nil,   // request
		&resp, // response
	)
	return
}

// Updates the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: KafkaConnections.
func (c *Client) UpdateKafkaConnection(name string, reqBody KafkaConnectionUpsertRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPut,
		fmt.Sprintf("/api/v1/connection/connection-templates/Kafka/connections/%s", name),
		reqBody, // request
		&resp,   // response
	)
	return
}

// Deletes the connection.
// Parameters:
// - name: The name of the connection.
// Tags: KafkaConnections.
func (c *Client) DeleteKafkaConnection(name string) (err error) {
	err = c.do(
		http.MethodDelete,
		fmt.Sprintf("/api/v1/connection/connection-templates/Kafka/connections/%s", name),
		nil, // request
		nil, // response
	)
	return
}

// Returns the list of available connections.
// Tags: KerberosConnections.
func (c *Client) ListKerberosConnections() (resp []ConnectionSummaryResponse, err error) {
	err = c.do(
		http.MethodGet,
		"/api/v1/connection/connection-templates/Kerberos/connections",
		nil,   // request
		&resp, // response
	)
	return
}

// Adds a new connection.
// Tags: KerberosConnections.
func (c *Client) CreateKerberosConnection(reqBody KerberosConnectionAddRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/Kerberos/connections",
		reqBody, // request
		&resp,   // response
	)
	return
}

// Validates the connection.
// Tags: KerberosConnections.
func (c *Client) TestKerberosConnection(reqBody KerberosConnectionTestRequest) (err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/Kerberos/connections/test",
		reqBody, // request
		nil,     // response
	)
	return
}

// Returns the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: KerberosConnections.
func (c *Client) GetKerberosConnection(name string) (resp KerberosConnectionResponse, err error) {
	err = c.do(
		http.MethodGet,
		fmt.Sprintf("/api/v1/connection/connection-templates/Kerberos/connections/%s", name),
		nil,   // request
		&resp, // response
	)
	return
}

// Updates the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: KerberosConnections.
func (c *Client) UpdateKerberosConnection(name string, reqBody KerberosConnectionUpsertRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPut,
		fmt.Sprintf("/api/v1/connection/connection-templates/Kerberos/connections/%s", name),
		reqBody, // request
		&resp,   // response
	)
	return
}

// Deletes the connection.
// Parameters:
// - name: The name of the connection.
// Tags: KerberosConnections.
func (c *Client) DeleteKerberosConnection(name string) (err error) {
	err = c.do(
		http.MethodDelete,
		fmt.Sprintf("/api/v1/connection/connection-templates/Kerberos/connections/%s", name),
		nil, // request
		nil, // response
	)
	return
}

// Returns the list of available connections.
// Tags: SchemaRegistryConnections.
func (c *Client) ListSchemaRegistryConnections() (resp []ConnectionSummaryResponse, err error) {
	err = c.do(
		http.MethodGet,
		"/api/v1/connection/connection-templates/SchemaRegistry/connections",
		nil,   // request
		&resp, // response
	)
	return
}

// Adds a new connection.
// Tags: SchemaRegistryConnections.
func (c *Client) CreateSchemaRegistryConnection(reqBody SchemaRegistryConnectionAddRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/SchemaRegistry/connections",
		reqBody, // request
		&resp,   // response
	)
	return
}

// Validates the connection.
// Tags: SchemaRegistryConnections.
func (c *Client) TestSchemaRegistryConnection(reqBody SchemaRegistryConnectionTestRequest) (err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/SchemaRegistry/connections/test",
		reqBody, // request
		nil,     // response
	)
	return
}

// Returns the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: SchemaRegistryConnections.
func (c *Client) GetSchemaRegistryConnection(name string) (resp SchemaRegistryConnectionResponse, err error) {
	err = c.do(
		http.MethodGet,
		fmt.Sprintf("/api/v1/connection/connection-templates/SchemaRegistry/connections/%s", name),
		nil,   // request
		&resp, // response
	)
	return
}

// Updates the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: SchemaRegistryConnections.
func (c *Client) UpdateSchemaRegistryConnection(name string, reqBody SchemaRegistryConnectionUpsertRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPut,
		fmt.Sprintf("/api/v1/connection/connection-templates/SchemaRegistry/connections/%s", name),
		reqBody, // request
		&resp,   // response
	)
	return
}

// Deletes the connection.
// Parameters:
// - name: The name of the connection.
// Tags: SchemaRegistryConnections.
func (c *Client) DeleteSchemaRegistryConnection(name string) (err error) {
	err = c.do(
		http.MethodDelete,
		fmt.Sprintf("/api/v1/connection/connection-templates/SchemaRegistry/connections/%s", name),
		nil, // request
		nil, // response
	)
	return
}

// Returns the list of available connections.
// Tags: ZookeeperConnections.
func (c *Client) ListZookeeperConnections() (resp []ConnectionSummaryResponse, err error) {
	err = c.do(
		http.MethodGet,
		"/api/v1/connection/connection-templates/Zookeeper/connections",
		nil,   // request
		&resp, // response
	)
	return
}

// Adds a new connection.
// Tags: ZookeeperConnections.
func (c *Client) CreateZookeeperConnection(reqBody ZookeeperConnectionAddRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/Zookeeper/connections",
		reqBody, // request
		&resp,   // response
	)
	return
}

// Validates the connection.
// Tags: ZookeeperConnections.
func (c *Client) TestZookeeperConnection(reqBody ZookeeperConnectionTestRequest) (err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connection-templates/Zookeeper/connections/test",
		reqBody, // request
		nil,     // response
	)
	return
}

// Returns the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: ZookeeperConnections.
func (c *Client) GetZookeeperConnection(name string) (resp ZookeeperConnectionResponse, err error) {
	err = c.do(
		http.MethodGet,
		fmt.Sprintf("/api/v1/connection/connection-templates/Zookeeper/connections/%s", name),
		nil,   // request
		&resp, // response
	)
	return
}

// Updates the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: ZookeeperConnections.
func (c *Client) UpdateZookeeperConnection(name string, reqBody ZookeeperConnectionUpsertRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPut,
		fmt.Sprintf("/api/v1/connection/connection-templates/Zookeeper/connections/%s", name),
		reqBody, // request
		&resp,   // response
	)
	return
}

// Deletes the connection.
// Parameters:
// - name: The name of the connection.
// Tags: ZookeeperConnections.
func (c *Client) DeleteZookeeperConnection(name string) (err error) {
	err = c.do(
		http.MethodDelete,
		fmt.Sprintf("/api/v1/connection/connection-templates/Zookeeper/connections/%s", name),
		nil, // request
		nil, // response
	)
	return
}

// Lists all connections templates. A connection's template defines the type of
// the connection, and the schema of it'sconfiguration..
// Tags: Connections.
func (c *Client) ListConnectionTemplates() (resp []ConnectionTemplateResponse, err error) {
	err = c.do(
		http.MethodGet,
		"/api/v1/connection/connection-templates",
		nil,   // request
		&resp, // response
	)
	return
}

// Returns the list of available connections.
// Tags: Connections.
func (c *Client) ListConnections() (resp []ConnectionSummaryResponse, err error) {
	err = c.do(
		http.MethodGet,
		"/api/v1/connection/connections",
		nil,   // request
		&resp, // response
	)
	return
}

// Returns the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: Connections.
func (c *Client) GetConnection1(name string) (resp ConnectionJsonResponse, err error) {
	err = c.do(
		http.MethodGet,
		fmt.Sprintf("/api/v1/connection/connections/%s", name),
		nil,   // request
		&resp, // response
	)
	return
}

// Validates the connection.
// Tags: Connections.
func (c *Client) TestConnection(reqBody TestConnectionAPIRequest) (err error) {
	err = c.do(
		http.MethodPost,
		"/api/v1/connection/connections/test",
		reqBody, // request
		nil,     // response
	)
	return
}

// Updates the connection details.
// Parameters:
// - name: The name of the connection.
// Tags: Connections.
func (c *Client) UpdateConnection1(name string, reqBody UpsertConnectionAPIRequest) (resp AddConnectionResponse, err error) {
	err = c.do(
		http.MethodPut,
		fmt.Sprintf("/api/v1/connection/connections/%s", name),
		reqBody, // request
		&resp,   // response
	)
	return
}

// Deletes the connection.
// Parameters:
// - name: The name of the connection.
// Tags: Connections.
func (c *Client) DeleteConnection1(name string) (err error) {
	err = c.do(
		http.MethodDelete,
		fmt.Sprintf("/api/v1/connection/connections/%s", name),
		nil, // request
		nil, // response
	)
	return
}

// do is an adapter between the method the generated code expects and the one
// that's being used here.
func (c *Client) do(method, path string, in, out interface{}) error {
	var body []byte
	if in != nil {
		var err error
		if body, err = json.Marshal(in); err != nil {
			return fmt.Errorf("marshal request body: %w", err)
		}
	}
	resp, err := c.Do(
		method,
		path,
		contentTypeJSON,
		body,
	)
	if err != nil {
		return fmt.Errorf("do api request: %w", err)
	}
	defer resp.Body.Close()
	if out == nil {
		return nil
	}
	if err := c.ReadJSON(resp, out); err != nil {
		return fmt.Errorf("read json: %w", err)
	}
	return nil
}
