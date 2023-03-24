package connection

import (
	"fmt"
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlueUpdateMinimal(t *testing.T) {
	m := genConnMock{}
	c := NewGlueGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"update", "the-name", "--aws-connection", "test-aws-conn", "--glue-registry-arn", "test-arn"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, ptrTo("the-name"), m.upName)
	assert.Equal(t, &api.UpsertConnectionAPIRequestV2{
		Configuration: configurationObjectAWSGlueSchemaRegistryV2{
			GlueRegistryArn: v2val[string]{"test-arn"},
			AccessKeyId:     v2ref{"test-aws-conn"},
			SecretAccessKey: v2ref{"test-aws-conn"},
		},
		Tags:         []string{},
		TemplateName: ptrTo("AWSGlueSchemaRegistry"),
	}, m.upReqV2)
}

func TestGlueUpdateExtraBells(t *testing.T) {
	m := genConnMock{}
	c := NewGlueGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"update", "the-name", "--aws-connection", "test-aws-conn", "--glue-registry-arn", "test-arn", "--glue-registry-cache-ttl", "123", "--glue-registry-cache-size", "31337", "--tags", "okay", "--tags", "cool"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, ptrTo("the-name"), m.upName)
	assert.Equal(t, &api.UpsertConnectionAPIRequestV2{
		Configuration: configurationObjectAWSGlueSchemaRegistryV2{
			GlueRegistryArn:       v2val[string]{"test-arn"},
			AccessKeyId:           v2ref{"test-aws-conn"},
			SecretAccessKey:       v2ref{"test-aws-conn"},
			GlueRegistryCacheTtl:  &v2val[int]{123},
			GlueRegistryCacheSize: &v2val[int]{31337},
		},
		Tags:         []string{"okay", "cool"},
		TemplateName: ptrTo("AWSGlueSchemaRegistry"),
	}, m.upReqV2)
}

func TestGlueTest(t *testing.T) {
	// The update flag is a "tri-state boolean", let's cover three cases.
	for name, upd := range map[string]*bool{"True": ptrTo(true), "False": ptrTo(false), "Absent": nil} {
		t.Run("UpdateFlag"+name, func(t *testing.T) {
			m := genConnMock{}
			c := NewGlueGroupCommand(&m, noUpload(t))
			args := []string{"test", "the-name", "--aws-connection", "test-aws-conn", "--glue-registry-arn", "test-arn"}
			if upd != nil {
				args = append(args, fmt.Sprintf("--update=%t", *upd))
			}
			c.SetArgs(args)
			err := c.Execute()
			require.NoError(t, err)
			assert.Equal(t, &api.TestConnectionAPIRequestV2{
				Name: "the-name",
				Configuration: configurationObjectAWSGlueSchemaRegistryV2{
					GlueRegistryArn: v2val[string]{"test-arn"},
					AccessKeyId:     v2ref{"test-aws-conn"},
					SecretAccessKey: v2ref{"test-aws-conn"},
				},
				TemplateName: "AWSGlueSchemaRegistry",
				Update:       upd,
			}, m.testReqV2)
		})
	}
}
