package api

type ConfigurationObjectElasticsearch struct {
	Nodes    []string `json:"nodes"`              // Required. The nodes of the Elasticsearch cluster to connect to, e.g. https://hostname:port. Use the tab key to specify multiple nodes.
	Password *string  `json:"password,omitempty"` // Optional. The password to connect to the Elasticsearch service.
	User     *string  `json:"user,omitempty"`     // Optional. The username to connect to the Elasticsearch service.
}

type ConfigurationObjectDataDogSite string

const (
	ConfigurationObjectDataDogSiteEU ConfigurationObjectDataDogSite = "EU"
	ConfigurationObjectDataDogSiteUS ConfigurationObjectDataDogSite = "US"
)

type ConfigurationObjectDataDog struct {
	APIKey         string                         `json:"apiKey"`                   // Required. The Datadog API key.
	Site           ConfigurationObjectDataDogSite `json:"site"`                     // Required. The Datadog site.
	ApplicationKey *string                        `json:"applicationKey,omitempty"` // Optional. The Datadog application key.
}

type ConfigurationObjectPagerDuty struct {
	IntegrationKey string `json:"integrationKey"` // Required. An Integration Key for PagerDuty's service with Events API v2 integration type.
}

type ConfigurationObjectPostgreSQLSslMode string

const (
	ConfigurationObjectPostgreSQLSslModeAllow      ConfigurationObjectPostgreSQLSslMode = "allow"
	ConfigurationObjectPostgreSQLSslModeDisable    ConfigurationObjectPostgreSQLSslMode = "disable"
	ConfigurationObjectPostgreSQLSslModePrefer     ConfigurationObjectPostgreSQLSslMode = "prefer"
	ConfigurationObjectPostgreSQLSslModeRequire    ConfigurationObjectPostgreSQLSslMode = "require"
	ConfigurationObjectPostgreSQLSslModeVerifyCa   ConfigurationObjectPostgreSQLSslMode = "verify-ca"
	ConfigurationObjectPostgreSQLSslModeVerifyFull ConfigurationObjectPostgreSQLSslMode = "verify-full"
)

type ConfigurationObjectPostgreSQL struct {
	Database string                               `json:"database"`           // Required. The database to connect to.
	Host     string                               `json:"host"`               // Required. The Postgres hostname.
	Port     int                                  `json:"port"`               // Required. The port number.
	SslMode  ConfigurationObjectPostgreSQLSslMode `json:"sslMode"`            // Required. The SSL connection mode as detailed in https://jdbc.postgresql.org/documentation/head/ssl-client.html.
	Username string                               `json:"username"`           // Required. The user name.
	Password *string                              `json:"password,omitempty"` // Optional. The password.
}

type ConfigurationObjectPrometheusAlertmanager struct {
	Endpoints []string `json:"endpoints"` // Required. Comma separated list of Alert Manager endpoints.
}

type ConfigurationObjectSlack struct {
	WebhookURL string `json:"webhookUrl"` // Required. The Slack endpoint to send the alert to.
}

type ConfigurationObjectSplunk struct {
	Host     string `json:"host"`           // Required. The host name for the HTTP Event Collector API of the Splunk instance.
	Insecure bool   `json:"insecure"`       // Required. This is not encouraged but is required for a Splunk Cloud Trial instance.
	Token    string `json:"token"`          // Required. HTTP event collector authorization token.
	UseHTTPs bool   `json:"useHttps"`       // Required. Use SSL.
	Port     *int   `json:"port,omitempty"` // Optional. The port number for the HTTP Event Collector API of the Splunk instance.
}

type ConfigurationObjectWebhook struct {
	Host     string   `json:"host"`            // Required. The host name.
	UseHTTPs bool     `json:"useHttps"`        // Required. Set to true in order to set the URL scheme to `https`. Will otherwise default to `http`.
	Creds    []string `json:"creds,omitempty"` // Optional. An array of (secret) strings to be passed over to alert channel plugins.
	Port     *int     `json:"port,omitempty"`  // Optional. An optional port number to be appended to the the hostname.
}
