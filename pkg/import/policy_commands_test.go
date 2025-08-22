package imports

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	test "github.com/lensesio/lenses-go/v5/test"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

const existingPoliciesResponse = `[
	{
	  "id": "test-policy-id-1",
	  "name": "omnihuborders-KFKDSP-26018-2",
	  "category": "PII",
	  "impact": {
		"apps": [],
		"connectors": [],
		"processors": [],
		"topics": ["some-topic"]
	  },
	  "impactType": "HIGH",
	  "obfuscation": "All",
	  "datasets": ["old-dataset"],
	  "fields": ["old-field"],
	  "lastUpdated": "2018-12-01 16:00:01",
	  "lastUpdatedUser": "admin"
	}
]`

func TestImportPoliciesPreserveDatasets(t *testing.T) {
	// Setup mock server
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/protection/policy" {
			w.Write([]byte(existingPoliciesResponse))
			return
		}

		if r.Method == http.MethodPut && r.URL.Path == "/api/protection/policy/test-policy-id-1" {
			// Capture the update request to verify it contains the correct datasets
			var updateRequest api.DataPolicyUpdateRequest
			err := json.NewDecoder(r.Body).Decode(&updateRequest)
			assert.NoError(t, err)

			// Verify that datasets from YAML file are preserved
			assert.NotNil(t, updateRequest.Datasets)
			expectedDatasets := []string{
				"omnihuborders.order_service.event.order_export_internal",
				"omnihuborders.order_service.event.export_dlq",
				"omnihuborders.order_service.master.order-status-json",
				"omnihuborders.order_service.master.order-status-dlq",
				"omnihuborders.order_service.event.order_export_internal_hype",
				"omnihuborders.order_service.event.order_export_internal_nam_lam",
				"omnihuborders.order_service.event.export_dlq_nam_lam",
				"omnihuborders.order_service.event.order_export_internal_nam_lam_hype",
			}
			assert.Equal(t, expectedDatasets, *updateRequest.Datasets, "Datasets should match the YAML file content")

			// Verify other fields are also updated from YAML
			assert.Equal(t, "omnihuborders-KFKDSP-26018-2", updateRequest.Name)
			assert.Equal(t, "PII", updateRequest.Category)
			assert.Equal(t, "HIGH", updateRequest.ImpactType)
			assert.Equal(t, "All", updateRequest.Obfuscation)
			assert.Contains(t, updateRequest.Fields, "billingAddress.addressLine1")
			assert.Contains(t, updateRequest.Fields, "customer.email")

			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
	assert.NoError(t, err)
	config.Client = client

	// Create temporary directory and policy YAML file for testing
	tempDir, err := os.MkdirTemp("", "policy-import-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	// Create the policies subdirectory structure
	policiesDir := filepath.Join(tempDir, "policies")
	err = os.MkdirAll(policiesDir, 0755)
	assert.NoError(t, err)

	// Write the test policy YAML file
	policyContent := `name: omnihuborders-KFKDSP-26018-2
impactType: HIGH
category: PII
obfuscation: All
datasets:
  - omnihuborders.order_service.event.order_export_internal
  - omnihuborders.order_service.event.export_dlq
  - omnihuborders.order_service.master.order-status-json
  - omnihuborders.order_service.master.order-status-dlq
  - omnihuborders.order_service.event.order_export_internal_hype
  - omnihuborders.order_service.event.order_export_internal_nam_lam
  - omnihuborders.order_service.event.export_dlq_nam_lam
  - omnihuborders.order_service.event.order_export_internal_nam_lam_hype
fields:
  - billingAddress.addressLine1
  - billingAddress.addressLine2
  - billingAddress.city
  - billingAddress.phoneNumber
  - billingAddress.email
  - billingAddress.firstName
  - billingAddress.lastName
  - customer.email
  - customer.firstName
  - customer.lastName
  - customer.phoneNumber`

	policyFile := filepath.Join(policiesDir, "test-policy.yaml")
	err = os.WriteFile(policyFile, []byte(policyContent), 0644)
	assert.NoError(t, err)

	// Create command and run the import
	cmd := &cobra.Command{}
	cmd.Flags().Set("silent", "true")

	// Test the loadPolicies function directly
	err = loadPolicies(config.Client, cmd, policiesDir)
	assert.NoError(t, err)
}

func TestImportPoliciesCreateNew(t *testing.T) {
	// Test case for creating a new policy (no existing policy with same name)
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/protection/policy" {
			// Return empty array - no existing policies
			w.Write([]byte("[]"))
			return
		}

		if r.Method == http.MethodPost && r.URL.Path == "/api/protection/policy" {
			// Capture the create request
			var createRequest api.DataPolicyRequest
			err := json.NewDecoder(r.Body).Decode(&createRequest)
			assert.NoError(t, err)

			// Verify datasets are properly set for new policy
			assert.NotNil(t, createRequest.Datasets)
			expectedDatasets := []string{
				"omnihuborders.order_service.event.order_export_internal",
				"omnihuborders.order_service.event.export_dlq",
			}
			assert.Equal(t, expectedDatasets, *createRequest.Datasets)

			w.WriteHeader(http.StatusOK)
			return
		}

		w.WriteHeader(http.StatusNotFound)
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))
	assert.NoError(t, err)
	config.Client = client

	// Create temporary directory and policy YAML file for testing
	tempDir, err := os.MkdirTemp("", "policy-create-test")
	assert.NoError(t, err)
	defer os.RemoveAll(tempDir)

	policiesDir := filepath.Join(tempDir, "policies")
	err = os.MkdirAll(policiesDir, 0755)
	assert.NoError(t, err)

	// Write a new policy YAML file
	policyContent := `name: new-test-policy
impactType: HIGH
category: PII
obfuscation: All
datasets:
  - omnihuborders.order_service.event.order_export_internal
  - omnihuborders.order_service.event.export_dlq
fields:
  - customer.email`

	policyFile := filepath.Join(policiesDir, "new-policy.yaml")
	err = os.WriteFile(policyFile, []byte(policyContent), 0644)
	assert.NoError(t, err)

	// Create command and run the import
	cmd := &cobra.Command{}
	cmd.Flags().Set("silent", "true")

	// Test the loadPolicies function
	err = loadPolicies(config.Client, cmd, policiesDir)
	assert.NoError(t, err)
}
