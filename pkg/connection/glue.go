package connection

import (
	"github.com/lensesio/lenses-go/v5/pkg/api"
	cobra "github.com/spf13/cobra"
)

func NewGlueGroupCommand(gen genericConnectionClientV2, up uploadFunc) *cobra.Command {
	cmd := &cobra.Command{
		Use:              "glue",
		Short:            "Manage Lenses AWS Glue Schema Registry connections",
		SilenceErrors:    true,
		TraverseChildren: true,
	}
	cmd.AddCommand(newGenericAPICommand[pseudoConfigurationObjectAWSGlueSchemaRegistry](
		"AWSGlueSchemaRegistry",
		apiv1ontov2Adapter[configurationObjectAWSGlueSchemaRegistryV2]{gen},
		up,
		FlagMapperOpts{
			Descriptions: map[string]string{
				"AWSConnection":         "Name of the Lenses AWS connection to use.",
				"GlueRegistryArn":       "Amazon Resource Name (ARN) of the Glue schema registry that you want to connect to.",
				"GlueRegistryCacheSize": "Amount of entries Lenses keeps in its AWS Glue schema cache.",
				"GlueRegistryCacheTtl":  "Period in milliseconds that Lenses will be updating its schema cache from AWS Glue.",
			},
		})...)
	return cmd
}

// genericConnectionClientV2 provides an api v2 connection interface. Glue is
// currently the only (and hopefully last) connection that uses it.
type genericConnectionClientV2 interface {
	GetConnection1(name string) (resp api.ConnectionJsonResponse, err error)
	ListConnections() (resp []api.ConnectionSummaryResponse, err error)
	TestConnectionV2(reqBody api.TestConnectionAPIRequestV2) (err error)
	UpdateConnectionV2(name string, reqBody api.UpsertConnectionAPIRequestV2) (resp api.AddConnectionResponse, err error)
	DeleteConnection1(name string) (err error)
}

// pseudoConfigurationObjectAWSGlueSchemaRegistry is how we represent the
// parameters to our cli users by having corresponding flags mapped ont its
// fields. It's not a real API object. Prior to sending it over to the API, it
// gets translated into a ConfigurationObjectAWSGlueSchemaRegistryV2, which is
// far less ergonomic. All of this ain't pretty and I can only hope that we'll
// quickly forget about this V2 stuff.
type pseudoConfigurationObjectAWSGlueSchemaRegistry struct {
	AWSConnection         string // Required. Name of the referenced AWS connection.
	GlueRegistryArn       string `json:"glueRegistryArn"`                 // Required. Enter the Amazon Resource Name (ARN) of the Glue schema registry that you want to connect to.
	GlueRegistryCacheSize *int   `json:"glueRegistryCacheSize,omitempty"` // Optional.
	GlueRegistryCacheTtl  *int   `json:"glueRegistryCacheTtl,omitempty"`  // Optional. The period in milliseconds that Lenses will be updating its schema cache from AWS Glue.
}

// When the dust settles this + its helpers should be moved into the api
// package.
type configurationObjectAWSGlueSchemaRegistryV2 struct {
	GlueRegistryArn       v2val[string] `json:"glueRegistryArn"`                 // Required. Enter the Amazon Resource Name (ARN) of the Glue schema registry that you want to connect to.
	GlueRegistryCacheSize *v2val[int]   `json:"glueRegistryCacheSize,omitempty"` // Optional.
	GlueRegistryCacheTtl  *v2val[int]   `json:"glueRegistryCacheTtl,omitempty"`  // Optional. The period in milliseconds that Lenses will be updating its schema cache from AWS Glue.
	// GlueRegistryDefaultCompatibility *string `json:"glueRegistryDefaultCompatibility,omitempty"` // Optional.
	AccessKeyId     v2ref `json:"accessKeyId"`     // ref
	SecretAccessKey v2ref `json:"secretAccessKey"` // ref
}

// asV2Obj is used by the apiv1ontov2Adapter to convert this into a v2
// connection API object.
func (c pseudoConfigurationObjectAWSGlueSchemaRegistry) asV2Obj() configurationObjectAWSGlueSchemaRegistryV2 {
	return configurationObjectAWSGlueSchemaRegistryV2{
		GlueRegistryArn:       newVal(c.GlueRegistryArn),
		AccessKeyId:           v2ref{c.AWSConnection},
		SecretAccessKey:       v2ref{c.AWSConnection},
		GlueRegistryCacheSize: newOptVal(c.GlueRegistryCacheSize),
		GlueRegistryCacheTtl:  newOptVal(c.GlueRegistryCacheTtl),
	}
}

// The fields of a v2 connection config object are not direct values, but are
// objects that either contain a reference to a different connection to get the
// value from, or the value.
type v2ref struct {
	Reference string `json:"reference"`
}

type v2val[T any] struct {
	Value T `json:"value"`
}

func newVal[T any](v T) v2val[T] {
	return v2val[T]{v}
}

func newOptVal[T any](v *T) *v2val[T] {
	if v == nil {
		return nil
	}
	return &v2val[T]{Value: *v}
}
