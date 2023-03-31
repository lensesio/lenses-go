package provision

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const configOrder = `
connections:
  kafka:
    tags: []
    templateName: Kafka
    configurationObject: {}
  schema-registry:
    templateName: SchemaRegiststries
    tags: []
    configurationObject: {}
  glue:
    templateName: AWSGlueSchemaRegistry
    tags: []
    configurationObject: {}
  aws:
    templateName: AWS
    tags: []
    configurationObject: {}
`

func TestProvisionConnectionOrder(t *testing.T) {
	awsSeen := false
	mockAPI := &mockApiClient{
		updateConnectionV1: func(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
			if *reqBody.TemplateName != "AWS" {
				assert.True(t, awsSeen, "aws should go before potential dependees")
			}
			awsSeen = awsSeen || *reqBody.TemplateName == "AWS"
			return api.AddConnectionResponse{}, nil
		},
		updateConnectionV2: func(name string, reqBody api.UpsertConnectionAPIRequestV2) (resp api.AddConnectionResponse, err error) {
			assert.True(t, awsSeen, "aws should go before potential dependees")
			return api.AddConnectionResponse{}, nil
		},
	}
	err := provision([]byte(configOrder), mockAPI, &mockGetter{})
	require.NoError(t, err)
	assert.True(t, awsSeen)
}

const configGlueV2 = `
connections:
  glue:
    templateName: AWSGlueSchemaRegistry
    tags: []
    configurationObject: {}
`

func TestProvisionGlueV2(t *testing.T) {
	glueSeen := false
	mockAPI := &mockApiClient{
		updateConnectionV1: func(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
			assert.False(t, *reqBody.TemplateName == "AWSGlueSchemaRegistry", "glue should go via v2")
			return api.AddConnectionResponse{}, nil
		},
		updateConnectionV2: func(name string, reqBody api.UpsertConnectionAPIRequestV2) (resp api.AddConnectionResponse, err error) {
			assert.True(t, *reqBody.TemplateName == "AWSGlueSchemaRegistry", "only glue should go via v2")
			glueSeen = true
			return api.AddConnectionResponse{}, nil
		},
	}
	err := provision([]byte(configGlueV2), mockAPI, &mockGetter{})
	require.NoError(t, err)
	assert.True(t, glueSeen)
}

const configObjMap = `connections:
  test:
    tags: [a,b,c]
    templateName: Test
    configurationObject:
      arr:
        - a
        - b
      map: # The yaml v2 lib maps maps to map[interface{}]interface{} which is incompatible with json's marshal.
        hello: world
        tedious: true
      number: 123
`

func TestProvisionConfigToObjectMapping(t *testing.T) {
	mockAPI := &mockApiClient{
		updateConnectionV1: func(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
			assert.Equal(t, "test", name)
			assert.Equal(t, []string{"a", "b", "c"}, reqBody.Tags)
			assert.Equal(t, "Test", *reqBody.TemplateName)
			// Check the json marshalled result rather than walk the minefield
			// of possible types.
			j, _ := json.Marshal(reqBody.ConfigurationObject)
			assert.JSONEq(t, `{"arr":["a","b"],"map":{"hello":"world","tedious":true},"number":123}`, string(j))
			return api.AddConnectionResponse{}, nil
		},
	}
	err := provision([]byte(configObjMap), mockAPI, &mockGetter{})
	require.NoError(t, err)
}

const configInlineFileref = `connections:
  test:
    templateName: T
    configurationObject:
      x:
        fileRef:
          inline: hello cruel world
`

func TestFileRefInlineNoB64(t *testing.T) {
	id := uuid.New()
	updated := false
	mockAPI := &mockApiClient{
		uploadFileFromReader: func(fileName string, r io.Reader) (uuid.UUID, error) {
			bs, _ := io.ReadAll(r)
			assert.Equal(t, "hello cruel world", string(bs))
			return id, nil
		},
		updateConnectionV1: func(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
			// Check the json marshalled result rather than walk the minefield
			// of possible types.
			j, _ := json.Marshal(reqBody.ConfigurationObject)
			assert.JSONEq(t, `{"x":{"fileId":"`+id.String()+`"}}`, string(j))
			updated = true
			return api.AddConnectionResponse{}, nil
		},
	}
	err := provision([]byte(configInlineFileref), mockAPI, &mockGetter{})
	require.NoError(t, err)
	assert.True(t, updated)
}

const configInlineFilerefB64 = `connections:
  test:
    templateName: T
    configurationObject:
      x:
        fileRef:
          inline: aGVsbG8gY3J1ZWwgd29ybGQ=
`

func TestFileRefInlineB64(t *testing.T) {
	id := uuid.New()
	updated := false
	mockAPI := &mockApiClient{
		uploadFileFromReader: func(fileName string, r io.Reader) (uuid.UUID, error) {
			bs, _ := io.ReadAll(r)
			assert.Equal(t, "hello cruel world", string(bs))
			return id, nil
		},
		updateConnectionV1: func(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
			// Check the json marshalled result rather than walk the minefield
			// of possible types.
			j, _ := json.Marshal(reqBody.ConfigurationObject)
			assert.JSONEq(t, `{"x":{"fileId":"`+id.String()+`"}}`, string(j))
			updated = true
			return api.AddConnectionResponse{}, nil
		},
	}
	err := provision([]byte(configInlineFilerefB64), mockAPI, &mockGetter{})
	require.NoError(t, err)
	assert.True(t, updated)
}

const configURLFileref = `connections:
  test:
    templateName: T
    configurationObject:
      x:
        fileRef:
          url: https://example.com/123
`

func TestFileRefURL(t *testing.T) {
	id := uuid.New()
	updated := false
	getter := &mockGetter{
		get: func(url string) (resp *http.Response, err error) {
			assert.Equal(t, "https://example.com/123", url)
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader("i am a remote file")),
			}, nil
		},
	}
	mockAPI := &mockApiClient{
		uploadFileFromReader: func(fileName string, r io.Reader) (uuid.UUID, error) {
			bs, _ := io.ReadAll(r)
			assert.Equal(t, "https://example.com/123", fileName)
			assert.Equal(t, "i am a remote file", string(bs))
			return id, nil
		},
		updateConnectionV1: func(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
			// Check the json marshalled result rather than walk the minefield
			// of possible types.
			j, _ := json.Marshal(reqBody.ConfigurationObject)
			assert.JSONEq(t, `{"x":{"fileId":"`+id.String()+`"}}`, string(j))
			updated = true
			return api.AddConnectionResponse{}, nil
		},
	}
	err := provision([]byte(configURLFileref), mockAPI, getter)
	require.NoError(t, err)
	assert.True(t, updated)
}

const configFileRefFilepath = `connections:
  test:
    templateName: T
    configurationObject:
      x:
        fileRef:
          filepath: ./testing/my-file.txt
`

func TestFileRefFilepath(t *testing.T) {
	id := uuid.New()
	updated := false
	mockAPI := &mockApiClient{
		uploadFileFromReader: func(fileName string, r io.Reader) (uuid.UUID, error) {
			bs, _ := io.ReadAll(r)
			assert.Equal(t, "my-file.txt", fileName)
			assert.Equal(t, "hello", string(bs))
			return id, nil
		},
		updateConnectionV1: func(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
			// Check the json marshalled result rather than walk the minefield
			// of possible types.
			j, _ := json.Marshal(reqBody.ConfigurationObject)
			assert.JSONEq(t, `{"x":{"fileId":"`+id.String()+`"}}`, string(j))
			updated = true
			return api.AddConnectionResponse{}, nil
		},
	}
	err := provision([]byte(configFileRefFilepath), mockAPI, &mockGetter{})
	require.NoError(t, err)
	assert.True(t, updated)
}

const configLicence = `
license:
  fileRef:
    inline: '{"key":"test"}'
`

func TestProvisionLicence(t *testing.T) {
	updated := false
	mockAPI := &mockApiClient{
		updateLicense: func(license api.License) error {
			assert.Equal(t, "test", license.Key)
			updated = true
			return nil
		},
	}
	err := provision([]byte(configLicence), mockAPI, &mockGetter{})
	require.NoError(t, err)
	assert.True(t, updated)
}

const configLicenceFilePath = `
license:
  fileRef:
    filePath: ./testing/my-lic.json # Note the casing of "Path".
`

// TestProvisionLicenceFilePath tests that provision.yamls with a license
// fileRef with a filePath key -- note the uppercase P that's not in the
// struct's field's tag -- are processed correctly.
func TestProvisionLicenceFilePath(t *testing.T) {
	updated := false
	mockAPI := &mockApiClient{
		updateLicense: func(license api.License) error {
			assert.Equal(t, "test", license.Key)
			updated = true
			return nil
		},
	}
	err := provision([]byte(configLicenceFilePath), mockAPI, &mockGetter{})
	require.NoError(t, err)
	assert.True(t, updated)
}

func TestErrorScenario(t *testing.T) {
	type probe struct {
		name string
		y    string
		a    mockApiClient
		g    mockGetter
	}
	ps := []probe{
		{
			name: "NonexistentFile",
			y:    strings.ReplaceAll(configFileRefFilepath, ".txt", ".invalid"),
		},
		{
			name: "BadURL",
			y:    strings.ReplaceAll(configURLFileref, "https", ""),
		},
		{
			name: "HTTPNetworkError",
			y:    configURLFileref,
			g:    mockGetter{func(url string) (resp *http.Response, err error) { return nil, errors.New("induced error") }},
		},
		{
			name: "HTTPNon2xx",
			y:    configURLFileref,
			g: mockGetter{func(url string) (resp *http.Response, err error) {
				return &http.Response{StatusCode: 404, Body: io.NopCloser(&bytes.Reader{})}, nil
			}},
		},
		{
			name: "APIUploadError",
			y:    configFileRefFilepath,
			a:    mockApiClient{uploadFileFromReader: func(fileName string, r io.Reader) (uuid.UUID, error) { return uuid.UUID{}, errors.New("induced error") }},
		},
		{
			name: "MissingTemplateName",
			y:    strings.ReplaceAll(configGlueV2, "templateName", "different"),
		},
		{
			name: "MissingConfObj",
			y:    strings.ReplaceAll(configGlueV2, "configurationObject", "different"),
		},
		{
			name: "BrokenUpdateConnectionV1",
			y:    configOrder,
			a: mockApiClient{updateConnectionV1: func(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
				return api.AddConnectionResponse{}, errors.New("induced error")
			}},
		},
		{
			name: "BrokenUpdateConnectionV2",
			y:    configOrder,
			a: mockApiClient{updateConnectionV2: func(name string, reqBody api.UpsertConnectionAPIRequestV2) (resp api.AddConnectionResponse, err error) {
				return api.AddConnectionResponse{}, errors.New("induced error")
			}},
		},
		{
			name: "BrokenUpdateLicense",
			y:    configLicence,
			a:    mockApiClient{updateLicense: func(license api.License) error { return errors.New("induced error") }},
		},
		{
			name: "BrokenUpdateLicense",
			y:    strings.ReplaceAll(configLicence, "key", `"`),
		},
	}
	for _, p := range ps {
		t.Run(p.name, func(t *testing.T) {
			err := provision([]byte(p.y), p.a, p.g)
			assert.Error(t, err, "expected this to error")
			t.Logf("As expected, we got an error:\n%s", err) // let a human judge how readable the message is.
		})
	}
}

type mockApiClient struct {
	uploadFileFromReader func(fileName string, r io.Reader) (uuid.UUID, error)
	updateLicense        func(license api.License) error
	updateConnectionV1   func(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error)
	updateConnectionV2   func(name string, reqBody api.UpsertConnectionAPIRequestV2) (resp api.AddConnectionResponse, err error)
}

func (m mockApiClient) UploadFileFromReader(fileName string, r io.Reader) (uuid.UUID, error) {
	if m.uploadFileFromReader == nil {
		return uuid.Nil, nil
	}
	return m.uploadFileFromReader(fileName, r)
}

func (m mockApiClient) UpdateLicense(license api.License) error {
	if m.updateLicense == nil {
		return nil
	}
	return m.updateLicense(license)
}

func (m mockApiClient) UpdateConnectionV1(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error) {
	if m.updateConnectionV1 == nil {
		return api.AddConnectionResponse{}, nil
	}
	return m.updateConnectionV1(name, reqBody)
}

func (m mockApiClient) UpdateConnectionV2(name string, reqBody api.UpsertConnectionAPIRequestV2) (resp api.AddConnectionResponse, err error) {
	if m.updateConnectionV2 == nil {
		return api.AddConnectionResponse{}, nil
	}
	return m.updateConnectionV2(name, reqBody)
}

type mockGetter struct {
	get func(url string) (resp *http.Response, err error)
}

func (m mockGetter) Get(url string) (resp *http.Response, err error) {
	if m.get == nil {
		return &http.Response{}, nil
	}
	return m.get(url)
}
