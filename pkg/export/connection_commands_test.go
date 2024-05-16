package export

import (
	"errors"
	"fmt"
	"io/fs"
	"testing"

	"github.com/lensesio/lenses-go/v5/test"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const provisioningFileName string = "provisioning.yaml"

func TestProvisionConnectionsCommand(t *testing.T) {
	t.Log("checks version 2 flag works")

	const provisioningStateResponse = "connections state response"

	mockAPI := &mockApiClient{
		getConnectionsState: func() (resp string, err error) {
			return provisioningStateResponse, nil
		},
	}

	mockWriter := mockFileWriter{
		mkdirAll: func(landscape string, mode fs.FileMode) error {
			var expectedMode fs.FileMode = 0o755
			assert.Equal(t, ".", landscape)
			assert.Equal(t, expectedMode, mode)

			return nil
		},
		writeFile: func(filePath string, content string, mode fs.FileMode) error {
			var expectedMode fs.FileMode = 0644
			assert.Equal(t, "provisioning.yaml", filePath)
			assert.Equal(t, expectedMode, mode)
			assert.Equal(t, provisioningStateResponse, content)

			return nil
		},
	}

	cmd := NewExportConnectionsCommand(mockAPI, mockWriter)
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	_, err := test.ExecuteCommand(cmd, "--version=2")

	require.NoError(t, err)
}

func TestProvisionConnectionsDirFlagCommand(t *testing.T) {
	t.Log("checks --dir flag works")

	const provisioningStateResponse = "connections state response"
	const outputFolder = "custom-folder"

	mockAPI := &mockApiClient{
		getConnectionsState: func() (resp string, err error) {
			return provisioningStateResponse, nil
		},
	}

	mockWriter := mockFileWriter{
		mkdirAll: func(landscape string, mode fs.FileMode) error {
			var expectedMode fs.FileMode = 0o755
			assert.Equal(t, outputFolder, landscape)
			assert.Equal(t, expectedMode, mode)

			return nil
		},
		writeFile: func(filePath string, content string, mode fs.FileMode) error {
			var expectedMode fs.FileMode = 0644
			assert.Equal(t, fmt.Sprintf("%s/provisioning.yaml", outputFolder), filePath)
			assert.Equal(t, expectedMode, mode)
			assert.Equal(t, provisioningStateResponse, content)

			return nil
		},
	}

	cmd := NewExportConnectionsCommand(mockAPI, mockWriter)
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	_, err := test.ExecuteCommand(cmd, "--version=2", fmt.Sprintf("--dir=%s", outputFolder))

	require.NoError(t, err)
}

func TestProvisionConnectionsNameFlagCommand(t *testing.T) {
	t.Log("checks --name flag is ignored")
	const provisioningStateResponse = "connections state response"

	mockAPI := &mockApiClient{
		getConnectionsState: func() (resp string, err error) {
			return provisioningStateResponse, nil
		},
	}

	mockWriter := mockFileWriter{
		mkdirAll: func(landscape string, mode fs.FileMode) error {
			var expectedMode fs.FileMode = 0o755
			assert.Equal(t, ".", landscape)
			assert.Equal(t, expectedMode, mode)

			return nil
		},
		writeFile: func(filePath string, content string, mode fs.FileMode) error {
			var expectedMode fs.FileMode = 0644
			assert.Equal(t, "provisioning.yaml", filePath)
			assert.Equal(t, expectedMode, mode)
			assert.Equal(t, provisioningStateResponse, content)

			return nil
		},
	}

	cmd := NewExportConnectionsCommand(mockAPI, mockWriter)
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	_, err := test.ExecuteCommand(cmd, "--version=2", "--name=any-name")

	require.NoError(t, err)
}

func TestProvisionedConnectionsFailToFetchConnections(t *testing.T) {
	t.Log("checks error when GetConnections fails")

	const errorMessage = "boom!"

	mockWriter := mockFileWriter{}

	mockAPI := &mockApiClient{
		getConnectionsState: func() (resp string, err error) {
			return "", errors.New(errorMessage)
		},
	}

	cmd := NewExportConnectionsCommand(mockAPI, mockWriter)
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	_, err := test.ExecuteCommand(cmd, "--version=2")

	require.Error(t, err)
	assert.Equal(t, fmt.Errorf("connections export: %w", fmt.Errorf("fetch provisioning connections: %w", errors.New(errorMessage))), err)
}

func TestProvisionedConnectionsFailToCreateDir(t *testing.T) {
	t.Log("checks error when MkDirAll fails")

	const errorMessage = "impossible to create directory"

	mockWriter := mockFileWriter{
		mkdirAll: func(string, fs.FileMode) error {
			return errors.New(errorMessage)
		},
	}

	mockAPI := &mockApiClient{
		getConnectionsState: func() (resp string, err error) {
			return "provisionedConnections", nil
		},
	}

	cmd := NewExportConnectionsCommand(mockAPI, mockWriter)
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	_, err := test.ExecuteCommand(cmd, "--version=2")

	require.Error(t, err)
	assert.Equal(t, fmt.Errorf("connections export: %w", errors.New(errorMessage)), err)
}

func TestProvisionedConnectionsFailWriteFile(t *testing.T) {
	t.Log("checks error when WriteFile fails")

	const errorMessage = "impossible to write file"

	mockWriter := mockFileWriter{
		writeFile: func(string, string, fs.FileMode) error {
			return errors.New(errorMessage)
		},

		mkdirAll: func(string, fs.FileMode) error {
			return nil
		},
	}

	mockAPI := &mockApiClient{
		getConnectionsState: func() (resp string, err error) {
			return "provisionedConnections", nil
		},
	}

	cmd := NewExportConnectionsCommand(mockAPI, mockWriter)
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	_, err := test.ExecuteCommand(cmd, "--version=2")
	require.Error(t, err)
	assert.Equal(t, fmt.Errorf("connections export: %w", errors.New(errorMessage)), err)
}

type mockApiClient struct {
	getConnectionsState func() (string, error)
	getConnections      func() ([]api.ConnectionList, error)
	getConnection       func(string) (api.Connection, error)
}

type mockFileWriter struct {
	mkdirAll  func(string, fs.FileMode) error
	writeFile func(string, string, fs.FileMode) error
}

func (w mockFileWriter) MkdirAll(landscape string, mode fs.FileMode) error {
	return w.mkdirAll(landscape, mode)
}

func (w mockFileWriter) WriteFile(filePath string, content string, mode fs.FileMode) error {
	return w.writeFile(filePath, content, mode)
}

// Implement GetConnection method for mockApiClient
func (m *mockApiClient) GetConnection(connectionName string) (api.Connection, error) {
	return m.getConnection(connectionName)

}

// Implement GetConnection method for mockApiClient
func (m *mockApiClient) GetConnections() ([]api.ConnectionList, error) {
	return m.getConnections()
}

// Implement GetConnection method for mockApiClient
func (m *mockApiClient) GetConnectionsState() (string, error) {
	return m.getConnectionsState()
}
