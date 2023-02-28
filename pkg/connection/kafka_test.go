package connection

import (
	"testing"

	"github.com/google/uuid"
	"github.com/lensesio/lenses-go/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKafkaList(t *testing.T) {
	m := kafkaMock{}
	c := NewKafkaGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"list"})
	err := c.Execute()
	require.NoError(t, err)
	assert.True(t, m.listCalled)
}

func TestKafkaUpdate(t *testing.T) {
	mockClient := kafkaMock{}
	mockUploader := uploadMock{}
	c := NewKafkaGroupCommand(&mockClient, mockUploader.upload)
	trustStoreID := uuid.New()
	c.SetArgs([]string{"update",
		"test-name",
		"--kafka-bootstrap-servers", "kbs0", "--kafka-bootstrap-servers", "kbs1",
		"--metrics-custom-url-mappings", "this=works", "--metrics-custom-url-mappings", "like=a-charm",
		"--metrics-port", "31337",
		"--tags", "tag0", "--tags", "tag1",
		"--metrics-ssl=false",
		"--keytab", "@my-keytab-file",
		"--ssl-truststore", trustStoreID.String(),
	})
	err := c.Execute()
	require.NoError(t, err)
	require.NotNil(t, mockClient.upName)
	assert.Equal(t, "test-name", *mockClient.upName)
	assert.NotEmpty(t, mockUploader["my-keytab-file"])
	assert.Equal(t, &api.KafkaConnectionUpsertRequest{
		ConfigurationObject: api.KafkaConnectionConfiguration{
			KafkaBootstrapServers:    []string{"kbs0", "kbs1"},
			MetricsCustomURLMappings: map[string]string{"this": "works", "like": "a-charm"},
			MetricsPort:              ptrTo(31337),
			MetricsSsl:               ptrTo(false),
			SslTruststore: &struct {
				FileId uuid.UUID `json:"fileId"`
			}{trustStoreID}, // flag as verbatim uuid.
			Keytab: &struct {
				FileId uuid.UUID `json:"fileId"`
			}{mockUploader["my-keytab-file"]}, // flag as @filename and having been mock "uploaded".
		},
		Tags: []string{"tag0", "tag1"},
	}, mockClient.upReq)
}

func TestKafkaGet(t *testing.T) {
	m := kafkaMock{}
	c := NewKafkaGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"get", "the-name"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, m.getName, ptrTo("the-name"))
}

func TestKafkaGetDefaultName(t *testing.T) {
	m := kafkaMock{}
	c := NewKafkaGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"get"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, m.getName, ptrTo("kafka"))
}

func TestKafkaDelete(t *testing.T) {
	m := kafkaMock{}
	c := NewKafkaGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"delete", "the-del-name"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, m.delName, ptrTo("the-del-name"))
}

func TestKafkaTest(t *testing.T) {
	mockClient := kafkaMock{}
	mockUploader := uploadMock{}
	c := NewKafkaGroupCommand(&mockClient, mockUploader.upload)
	trustStoreID := uuid.New()
	c.SetArgs([]string{"test",
		"test-name",
		"--kafka-bootstrap-servers", "kbs0", "--kafka-bootstrap-servers", "kbs1",
		"--metrics-custom-url-mappings", "this=works", "--metrics-custom-url-mappings", "like=a-charm",
		"--metrics-port", "31337",
		"--metrics-ssl=false",
		"--keytab", "@my-keytab-file",
		"--ssl-truststore", trustStoreID.String(),
		"--update=true",
	})
	err := c.Execute()
	require.NoError(t, err)
	assert.NotEmpty(t, mockUploader["my-keytab-file"])
	assert.Equal(t, &api.KafkaConnectionTestRequest{
		ConfigurationObject: api.KafkaConnectionConfiguration{
			KafkaBootstrapServers:    []string{"kbs0", "kbs1"},
			MetricsCustomURLMappings: map[string]string{"this": "works", "like": "a-charm"},
			MetricsPort:              ptrTo(31337),
			MetricsSsl:               ptrTo(false),
			SslTruststore: &struct {
				FileId uuid.UUID `json:"fileId"`
			}{trustStoreID}, // flag as verbatim uuid.
			Keytab: &struct {
				FileId uuid.UUID `json:"fileId"`
			}{mockUploader["my-keytab-file"]}, // flag as @filename and having been mock "uploaded".
		},
		Name:   "test-name",
		Update: ptrTo(true),
	}, mockClient.testReq)
}

func noUpload(t *testing.T) func(path string) (uuid.UUID, error) {
	return func(path string) (uuid.UUID, error) {
		t.Fatalf("no upload expected")
		return uuid.UUID{}, nil
	}
}

type uploadMock map[string]uuid.UUID

func (u uploadMock) upload(path string) (uuid.UUID, error) {
	id := uuid.New()
	u[path] = id
	return id, nil
}

type kafkaMock struct {
	getName *string
	getResp api.KafkaConnectionResponse

	upName *string
	upReq  *api.KafkaConnectionUpsertRequest
	upResp api.AddConnectionResponse

	delName *string

	testReq *api.KafkaConnectionTestRequest

	listCalled bool
	listResp   []api.ConnectionSummaryResponse

	genErr error
}

func (k *kafkaMock) GetKafkaConnection(name string) (resp api.KafkaConnectionResponse, err error) {
	k.getName = &name
	return k.getResp, k.genErr
}

func (k *kafkaMock) UpdateKafkaConnection(name string, reqBody api.KafkaConnectionUpsertRequest) (resp api.AddConnectionResponse, err error) {
	k.upName = &name
	k.upReq = &reqBody
	return k.upResp, k.genErr
}

func (k *kafkaMock) DeleteKafkaConnection(name string) (err error) {
	k.delName = &name
	return k.genErr
}

func (k *kafkaMock) TestKafkaConnection(reqBody api.KafkaConnectionTestRequest) (err error) {
	k.testReq = &reqBody
	return k.genErr
}

func (k *kafkaMock) ListKafkaConnections() (resp []api.ConnectionSummaryResponse, err error) {
	k.listCalled = true
	return k.listResp, k.genErr
}
