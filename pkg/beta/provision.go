package beta

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/lensesio/lenses-go/pkg"
	"github.com/lensesio/lenses-go/pkg/api"
	"github.com/mitchellh/mapstructure"
)

// Config reprents the high level structure of expected file for 'provision' cmd
type Config struct {
	License struct {
		FileRef FileRef `yaml:"fileRef"`
	} `yaml:"license"`
	Connections map[string]interface{} `yaml:"connections"`
}

// FileRef is structure that specifies how to retrieve the file to be uploaded
type FileRef struct {
	Inline   string `yaml:"inline"`
	URL      string `yaml:"URL"`
	Filepath string `yaml:"filepath"`
}

// FileUploadResp corresponds to the response from '/api/v1/files' endpoint
// used to capture the 'id'.
type FileUploadResp struct {
	ID         string `json:"id"`
	Filename   string `json:"filename"`
	Size       int    `json:"size"`
	UploadedBy string `json:"uploadedBy"`
}

func fileRefToBytes(fileRef FileRef) (fileAsBytes []byte, fileName string, err error) {
	switch {
	case fileRef.Filepath != "":
		if fileAsBytes, err = os.ReadFile(fileRef.Filepath); err != nil {
			return nil, "", err
		}

		fileName = filepath.Base(fileRef.Filepath)
		fmt.Printf("loaded file from '%s'\n", fileRef.Filepath)

	case fileRef.Inline != "":
		// inline value may be base64 encoded (connections) or not (license)
		if fileAsBytes, err = base64.StdEncoding.DecodeString(fileRef.Inline); err != nil {
			fileAsBytes = []byte(fileRef.Inline)
		}

		// if file contents are passed inline then use its SHA as unique filename
		sha256 := sha256.Sum256(fileAsBytes)
		fileName = fmt.Sprintf("%x", sha256)
		fmt.Printf("loaded file contents inline with SHA value '%s'\n", fileName)

	case fileRef.URL != "":
		// check that URL value is valid
		if _, err := url.ParseRequestURI(fileRef.URL); err != nil {
			return nil, "", err
		}

		resp, err := http.Get(fileRef.URL)
		if err != nil {
			return nil, "", err
		}
		defer resp.Body.Close()

		if fileAsBytes, err = ioutil.ReadAll(resp.Body); err != nil {
			return nil, "", err
		}

		fileName = fileRef.URL
		fmt.Printf("loaded file from URL '%s'\n", fileRef.URL)

	default:
		return nil, "", errors.New("did not find a valid file reference")
	}

	return fileAsBytes, fileName, nil
}

// HTTPDoer is the interface that both api.client and mock client use
type HTTPDoer interface {
	Do(method, path, contentType string, send []byte, options ...api.RequestOption) (*http.Response, error)
	ReadResponseBody(resp *http.Response) ([]byte, error)
}

// uploadFile accepts a file reference and httpdoer that knows how to upload a file
// and returns a file ID.
func uploadFile(fr interface{}, client HTTPDoer) (string, error) {
	var fileRef FileRef
	if err := mapstructure.Decode(fr, &fileRef); err != nil {
		return "", err
	}

	fileAsBytes, fileName, err := fileRefToBytes(fileRef)
	if err != nil {
		return "", err
	}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return "", err
	}

	if _, err := part.Write(fileAsBytes); err != nil {
		return "", err
	}

	writer.Close()
	fileUploadBody := body.Bytes()

	resp, err := client.Do(http.MethodPost, pkg.FileUploadPath, writer.FormDataContentType(), fileUploadBody)
	if err != nil {
		return "", err
	}
	message, err := client.ReadResponseBody(resp)
	if err != nil {
		return "", err
	}

	var fileResp FileUploadResp
	if err := json.Unmarshal(message, &fileResp); err != nil {
		return "", err
	}

	fmt.Printf("file successfully uploaded: \n%v\n", string(message))
	return fileResp.ID, nil
}

func parseConfig(input interface{}, client HTTPDoer) error {
	refs := gatherFileRefs(input)

	for _, ref := range refs {
		v := ref["fileRef"]
		fileID, err := uploadFile(v, client)
		if err != nil {
			return err
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

var errMissingLicence = errors.New("'license' key not found or empty value")
var errMissingConnections = errors.New("no connections found")
var errInvalidConnectionsStruct = errors.New("cannot decode Connections, expected a map")

// checkConfigValidity verifies the the config passed has the expected
// high level structure
func checkConfigValidity(config map[interface{}]interface{}) error {

	var conf Config
	if err := mapstructure.Decode(config, &conf); err != nil {
		return errInvalidConnectionsStruct
	}

	if conf.License.FileRef == (FileRef{}) {
		return errMissingLicence
	}

	if len(conf.Connections) == 0 {
		return errMissingConnections
	}
	return nil
}
