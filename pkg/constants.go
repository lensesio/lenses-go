package pkg

//ERRORS
const (
	ErrResourceNotFoundMessage      = 404
	ErrResourceNotAccessibleMessage = 403
	ErrResourceNotGoodMessage       = 400
	ErrResourceInternal             = 500
)

//Paths
const (
	SQLPath        = "apps/sql"
	ConnectorsPath = "apps/connectors"

	GroupsPath          = "groups"
	UsersPath           = "users"
	ServiceAccountsPath = "service-accounts"

	AclsPath   = "kafka/acls"
	TopicsPath = "kafka/topics"
	QuotasPath = "kafka/quotas"

	SchemasPath       = "schemas"
	AlertSettingsPath = "alert-settings"
	PoliciesPath      = "policies"

	ConnectionsFilePath        = "connections"
	ConnectionsAPIPath         = "v1/connection/connections"
	DatasetsAPIPath            = "v1/datasets"
	ConnectionTemplatesAPIPath = "v1/connection/connection-templates"
	ConsumersGroupPath         = "api/consumers"
	ElasticsearchIndexesPath   = "/api/elastic/indexes"
	AlertChannelsPath          = "api/v1/alert/channels"
	AlertChannelTemplatesPath  = "api/v1/alert/channel-templates"
	AuditChannelTemplatesPath  = "api/v1/audit/channel-templates"
	AlertsSettingsPath         = "api/v1/alert/settings"
	AlertsPathSSE              = "api/sse/alerts"
	AlertEventsPath            = "api/v1/alert/events"
	MetadataTopicsPath         = "api/v1/metadata/topics"

	LicensePath = "api/v1/license"
)
