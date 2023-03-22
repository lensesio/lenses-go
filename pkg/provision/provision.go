package provision

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/yaml.v2"
)

type apiClient interface {
	UploadFileFromReader(fileName string, r io.Reader) (uuid.UUID, error)
	UpdateLicense(license api.License) error
	UpdateConnectionV1(name string, reqBody api.UpsertConnectionAPIRequest) (resp api.AddConnectionResponse, err error)
	UpdateConnectionV2(name string, reqBody api.UpsertConnectionAPIRequestV2) (resp api.AddConnectionResponse, err error)
}

// httpGetter is used to download fileRefs that reference a url.
type httpGetter interface {
	Get(url string) (resp *http.Response, err error)
}

// Config represents the high level structure of expected file for 'provision'
// cmd.
type Config struct {
	License struct {
		FileRef FileRef `yaml:"fileRef"`
	} `yaml:"license"`
	Connections map[string]connection `yaml:"connections"`
}

// connection is effectively an api.UpsertConnectionAPIRequest - with yaml tags.
type connection struct {
	ConfigurationObject any      `yaml:"configurationObject"`
	Tags                []string `yaml:"tags"`
	TemplateName        string   `yaml:"templateName"`
}

// asAPIObjectV1 converts the connection into the equivalent API object, and
// massages any map[interface{}]interface{} (produced by unmarshalling yaml)
// into map[string]interface{} (required by stdlib's json marshal).
func (c connection) asAPIObjectV1() api.UpsertConnectionAPIRequest {
	return api.UpsertConnectionAPIRequest{
		ConfigurationObject: cleanupMapValue(c.ConfigurationObject),
		Tags:                c.Tags,
		TemplateName:        &c.TemplateName,
	}
}

func (c connection) asAPIObjectV2() api.UpsertConnectionAPIRequestV2 {
	return api.UpsertConnectionAPIRequestV2{
		Configuration: cleanupMapValue(c.ConfigurationObject),
		Tags:          c.Tags,
		TemplateName:  &c.TemplateName,
	}
}

// FileRef is structure that specifies how to retrieve the file to be uploaded
type FileRef struct {
	Inline   string `yaml:"inline"`
	URL      string `yaml:"URL"`
	Filepath string `yaml:"filepath"`
}

func fileRefToBytes(fileRef FileRef, getter httpGetter) (fileAsBytes []byte, fileName string, err error) {
	switch {
	case fileRef.Filepath != "":
		if fileAsBytes, err = os.ReadFile(fileRef.Filepath); err != nil {
			return nil, "", err
		}

		fileName = filepath.Base(fileRef.Filepath)
		fmt.Fprintf(os.Stderr, "loaded file from %q\n", fileRef.Filepath)

	case fileRef.Inline != "":
		// inline value may be base64 encoded (connections) or not (license)
		// (how could this ever go wrong...)
		if fileAsBytes, err = base64.StdEncoding.DecodeString(fileRef.Inline); err != nil {
			fileAsBytes = []byte(fileRef.Inline)
		}

		// if file contents are passed inline then use its SHA as unique filename
		sha256 := sha256.Sum256(fileAsBytes)
		fileName = fmt.Sprintf("%x", sha256)
		fmt.Fprintf(os.Stderr, "loaded file contents inline with SHA value %q\n", fileName)

	case fileRef.URL != "":
		// check that URL value is valid
		if _, err := url.ParseRequestURI(fileRef.URL); err != nil {
			return nil, "", err
		}

		resp, err := getter.Get(fileRef.URL)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, "", fmt.Errorf("get %q: received non-2xx status code: %d", fileRef.URL, resp.StatusCode)
		}

		if fileAsBytes, err = io.ReadAll(resp.Body); err != nil {
			return nil, "", err
		}

		fileName = fileRef.URL
		fmt.Fprintf(os.Stderr, "loaded file from URL %q\n", fileRef.URL)

	default:
		return nil, "", errors.New("did not find a valid file reference")
	}

	return fileAsBytes, fileName, nil
}

// uploadFileref accepts a file reference and httpdoer that knows how to upload a file
// and returns a file ID.
func uploadFileref(fr interface{}, client apiClient, getter httpGetter) (string, error) {
	var fileRef FileRef
	if err := mapstructure.Decode(fr, &fileRef); err != nil {
		return "", err
	}

	fileAsBytes, fileName, err := fileRefToBytes(fileRef, getter)
	if err != nil {
		return "", err
	}
	resp, err := client.UploadFileFromReader(fileName, bytes.NewReader(fileAsBytes))
	if err != nil {
		return "", err
	}
	fmt.Fprintf(os.Stderr, "uploaded file %q which got assigned id %q\n", fileName, resp.String())
	return resp.String(), nil
}

func patchFileRefs(conf Config, client apiClient, getter httpGetter) error {
	for name, c := range conf.Connections {
		if err := patchFileRef(c.ConfigurationObject, client, getter); err != nil {
			return fmt.Errorf("connection %q: %w", name, err)
		}
	}
	return nil
}

func patchFileRef(c any, client apiClient, getter httpGetter) error {
	for _, ref := range gatherFileRefs(c) {
		v := ref["fileRef"]
		fileID, err := uploadFileref(v, client, getter)
		if err != nil {
			return fmt.Errorf("upload file: %w", err)
		}
		delete(ref, "fileRef")
		ref["fileId"] = fileID
	}
	return nil
}

func gatherFileRefs(input interface{}) []map[interface{}]interface{} {
	// Inspired from from https://gist.github.com/niski84/a6a3b825b6704cc2cbfd39c97b89e640
	in, ok := input.(map[interface{}]interface{})
	if !ok {
		return nil
	}

	var out []map[interface{}]interface{}

	for k, v := range in {
		if k == "fileRef" {
			out = append(out, in)
		}

		m, valueIsMap := v.(map[interface{}]interface{})
		// We only want to replace 'fileRef' for connections
		if k != "license" && valueIsMap {
			out = append(out, gatherFileRefs(m)...)
		}

		if va, valueIsSliceOfInterfaces := v.([]interface{}); valueIsSliceOfInterfaces {
			for _, a := range va {
				out = append(out, gatherFileRefs(a)...)
			}
		}
	}

	return out
}

// checkConfigValidity verifies the the config passed has the expected
// high level structure (license is optional)
func checkConfigValidity(conf Config) error {
	for name, c := range conf.Connections {
		if c.TemplateName == "" {
			return fmt.Errorf("connection %q misses field: templateName", name)
		}
		if c.ConfigurationObject == nil {
			return fmt.Errorf("connection %q misses field: configurationObject", name)
		}
	}
	return nil
}

// provision func embeds the flow of provisioning logic
func provision(yamlFileAsBytes []byte, client apiClient, getter httpGetter) error {
	var conf Config
	if err := yaml.Unmarshal(yamlFileAsBytes, &conf); err != nil {
		return err
	}

	// Check if input file has the expected structure
	if err := checkConfigValidity(conf); err != nil {
		return err
	}

	// Parse config and replace 'fileRef' with 'fileId' only for connections
	if err := patchFileRefs(conf, client, getter); err != nil {
		return err
	}

	// Handle connections. The original idea was that the CLI would pass the
	// provision information verbatim to the API, not understanding its content.
	// That didn't age well, so here we are:
	// 1. "AWS" connections go first, because they can be referenced by other
	// connections;
	for name, conn := range conf.Connections {
		if conn.TemplateName != "AWS" {
			continue
		}
		if _, err := client.UpdateConnectionV1(name, conn.asAPIObjectV1()); err != nil {
			return fmt.Errorf("configure connection %q: %w", name, err)
		}
		fmt.Fprintf(os.Stderr, "Updated connection: %q\n", name)
	}
	// 2. "AWSGlueSchemaRegistry" connections go to v2 of the API.
	for name, conn := range conf.Connections {
		if conn.TemplateName == "AWS" { // Has already been done.
			continue
		}
		var err error
		if conn.TemplateName == "AWSGlueSchemaRegistry" {
			_, err = client.UpdateConnectionV2(name, conn.asAPIObjectV2())
		} else {
			_, err = client.UpdateConnectionV1(name, conn.asAPIObjectV1())
		}
		if err != nil {
			return fmt.Errorf("configure connection %q: %w", name, err)
		}
		fmt.Fprintf(os.Stderr, "Updated connection: %q\n", name)
	}

	// License is an optional value.
	if conf.License.FileRef != (FileRef{}) {
		licenseAsBytes, _, err := fileRefToBytes(conf.License.FileRef, getter)
		if err != nil {
			return fmt.Errorf("parse license reference: %w", err)
		}

		var lic api.License
		if err := json.Unmarshal(licenseAsBytes, &lic); err != nil {
			return fmt.Errorf("interpret license: %w", err)
		}

		if err := client.UpdateLicense(lic); err != nil {
			return fmt.Errorf("update license: %w", err)
		}
		fmt.Fprintln(os.Stderr, "license updated successfully")
	}
	return nil
}

// cleanup* have been borrowed from https://github.com/go-yaml/yaml/issues/139
func cleanupInterfaceArray(in []interface{}) []interface{} {
	res := make([]interface{}, len(in))
	for i, v := range in {
		res[i] = cleanupMapValue(v)
	}
	return res
}

// cleanup* have been borrowed from https://github.com/go-yaml/yaml/issues/139
func cleanupInterfaceMap(in map[interface{}]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range in {
		res[fmt.Sprintf("%v", k)] = cleanupMapValue(v)
	}
	return res
}

// cleanup* have been borrowed from https://github.com/go-yaml/yaml/issues/139
func cleanupMapValue(v interface{}) interface{} {
	switch v := v.(type) {
	case []interface{}:
		return cleanupInterfaceArray(v)
	case map[interface{}]interface{}:
		return cleanupInterfaceMap(v)
	default:
		return v
	}
}
