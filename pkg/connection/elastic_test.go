package connection

import (
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestElasticList(t *testing.T) {
	m := genConnMock{}
	c := NewElasticsearchGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"list"})
	err := c.Execute()
	require.NoError(t, err)
	assert.True(t, m.listCalled)
}

func TestElasticUpdate(t *testing.T) {
	m := genConnMock{}
	c := NewElasticsearchGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"update", "the-name", "--nodes", "x", "--nodes", "y", "--es-user", "u", "--es-password", "p"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, ptrTo("the-name"), m.upName)
	assert.Equal(t, &api.UpsertConnectionAPIRequest{
		ConfigurationObject: api.ConfigurationObjectElasticsearch{
			Nodes:    []string{"x", "y"},
			Password: ptrTo("p"),
			User:     ptrTo("u"),
		},
		Tags:         []string{},
		TemplateName: ptrTo("Elasticsearch"),
	}, m.upReq)
}

func TestElasticGet(t *testing.T) {
	m := genConnMock{}
	c := NewElasticsearchGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"get", "the-name"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, ptrTo("the-name"), m.getName)
}

func TestElasticGetNoDefault(t *testing.T) {
	m := genConnMock{}
	c := NewElasticsearchGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"get"})
	err := c.Execute()
	require.Error(t, err)
}

func TestElasticDelete(t *testing.T) {
	m := genConnMock{}
	c := NewElasticsearchGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"delete", "the-name"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, ptrTo("the-name"), m.delName)
}

func TestElasticTest(t *testing.T) {
	m := genConnMock{}
	c := NewElasticsearchGroupCommand(&m, noUpload(t))
	c.SetArgs([]string{"test", "the-name", "--nodes", "x", "--nodes", "y", "--es-user", "u", "--es-password", "p"})
	err := c.Execute()
	require.NoError(t, err)
	assert.Equal(t, &api.TestConnectionAPIRequest{
		Name: "the-name",
		ConfigurationObject: api.ConfigurationObjectElasticsearch{
			Nodes:    []string{"x", "y"},
			Password: ptrTo("p"),
			User:     ptrTo("u"),
		},
		TemplateName: "Elasticsearch",
	}, m.testReq)
}

type genConnMock struct {
	getName *string
	getResp api.ConnectionJsonResponse

	upName *string
	upReq  *api.UpsertConnectionAPIRequest
	upResp api.AddConnectionResponse

	delName *string

	testReq *api.TestConnectionAPIRequest

	listCalled bool
	listResp   []api.ConnectionSummaryResponse

	genErr error
}

func (g *genConnMock) GetConnection1(name string) (resp api.ConnectionJsonResponse, err error) {
	g.getName = &name
	return g.getResp, g.genErr
}
func (g *genConnMock) ListConnections() (resp []api.ConnectionSummaryResponse, err error) {
	g.listCalled = true
	return g.listResp, g.genErr
}
func (g *genConnMock) TestConnection(reqBody api.TestConnectionAPIRequest) (err error) {
	g.testReq = &reqBody
	return g.genErr
}
func (g *genConnMock) UpdateConnectionV1(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
	g.upName = &name
	g.upReq = &reqBody
	return g.upResp, g.genErr
}
func (g *genConnMock) DeleteConnection1(name string) (err error) {
	g.delName = &name
	return g.genErr
}
