package api

import (
	"bytes"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/lensesio/lenses-go/pkg"
)

type FileMetadataEntity struct {
	Filename    string    `json:"filename"`              // Required.
	ID          uuid.UUID `json:"id"`                    // Required.
	Size        int       `json:"size"`                  // Required.
	UploadedAt  time.Time `json:"uploadedAt"`            // Required.
	UploadedBy  string    `json:"uploadedBy"`            // Required.
	ContentType *string   `json:"contentType,omitempty"` // Optional.
}

func (c *Client) UploadFile(fileName string) (uuid.UUID, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return uuid.Nil, err
	}
	defer f.Close()
	return c.UploadFileFromReader(fileName, f)
}

func (c *Client) UploadFileFromReader(fileName string, r io.Reader) (uuid.UUID, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return uuid.Nil, err
	}
	if _, err := io.Copy(part, r); err != nil {
		return uuid.Nil, err
	}
	if err := writer.Close(); err != nil {
		return uuid.Nil, err
	}
	fileUploadBody := body.Bytes()

	resp, err := c.Do(http.MethodPost, pkg.FileUploadPath, writer.FormDataContentType(), fileUploadBody)
	if err != nil {
		return uuid.Nil, err
	}
	message, err := c.ReadResponseBody(resp)
	if err != nil {
		return uuid.Nil, err
	}

	var fileResp FileMetadataEntity
	if err := json.Unmarshal(message, &fileResp); err != nil {
		return uuid.Nil, err
	}
	return fileResp.ID, nil
}
