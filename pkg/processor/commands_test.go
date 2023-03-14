package processor

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/lensesio/lenses-go/v5/pkg/api"
	config "github.com/lensesio/lenses-go/v5/pkg/configs"
	test "github.com/lensesio/lenses-go/v5/test"
	"github.com/stretchr/testify/assert"
)

var aksk8s = api.KubernetesTarget{Cluster: "aks", Namespaces: []string{"prod"}, Version: "1.0.0"}
var aksk8Res = ListTargetsResult{Type: "Kubernetes", ClusterName: "aks", Namespace: "prod", Version: "1.0.0"}
var eksk8s = api.KubernetesTarget{Cluster: "eks", Namespaces: []string{"dev"}, Version: "1.0.0"}
var eksk8Res = ListTargetsResult{Type: "Kubernetes", ClusterName: "eks", Namespace: "dev", Version: "1.0.0"}
var connect = api.KafkaConnectTarget{Cluster: "my-kafka-connect", Version: "1.0.0"}
var connectRes = ListTargetsResult{Type: "Connect", ClusterName: "my-kafka-connect", Namespace: "", Version: "1.0.0"}
var targetList = &api.DeploymentTargets{
	Kubernetes: []api.KubernetesTarget{aksk8s, eksk8s},
	Connect:    []api.KafkaConnectTarget{connect},
}

var processorRegisteredResponse = "processor-id"
var targetsAsJSON, _ = json.Marshal(targetList)
var processorRegisteredAsJSON, _ = json.Marshal(processorRegisteredResponse)

func TestListTargetDeploymentCommand(t *testing.T) {

	list := [3]ListTargetsResult{aksk8Res, eksk8Res, connectRes}
	e, _ := json.Marshal(list)

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(string(targetsAsJSON)))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGetProcessorsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "targets")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)
	assert.Equal(t, string(e), strings.TrimSuffix(output, "\n"))

	config.Client = nil
}

func TestListTargetK8sDeploymentCommand(t *testing.T) {

	//result := `[{"type":"Kubernetes","clusterName":"aks","namespace":"prod","version":"1.0.0"},{"type":"Kubernetes","clusterName":"eks","namespace":"dev","version":"1.0.0"}]`
	list := [2]ListTargetsResult{aksk8Res, eksk8Res}
	e, _ := json.Marshal(list)

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(string(targetsAsJSON)))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGetProcessorsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "targets", "--target-type=kubernetes")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)
	assert.Equal(t, string(e), strings.TrimSuffix(output, "\n"))

	config.Client = nil
}

func TestListTargetK8sClusterNameDeploymentCommand(t *testing.T) {

	list := [1]ListTargetsResult{aksk8Res}
	e, _ := json.Marshal(list)

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(string(targetsAsJSON)))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGetProcessorsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "targets", "--target-type=kubernetes", "--cluster-name=aks")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)
	assert.Equal(t, string(e), strings.TrimSuffix(output, "\n"))

	config.Client = nil
}

func TestListTargetConnectDeploymentCommand(t *testing.T) {

	list := [1]ListTargetsResult{connectRes}
	e, _ := json.Marshal(list)

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(string(targetsAsJSON)))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewGetProcessorsCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")
	output, err := test.ExecuteCommand(cmd, "targets", "--target-type=connect")

	assert.Nil(t, err)
	assert.NotEmpty(t, output)
	assert.Equal(t, string(e), strings.TrimSuffix(output, "\n"))

	config.Client = nil
}

func TestNewProcessorCreateCommand(t *testing.T) {

	tests := map[string]struct {
		params   []string
		expected string
	}{
		"all_params": {
			params: []string{
				"--name=\"processor arBiTRary StrinG with 989784987  $#% 读写汉字 - 学中文 name 1\"",
				"--sql=\"SET defaults.topic.autocreate=true; INSERT INTO topic-test-again-2 SELECT STREAM * FROM telecom_italia;\"",
				"--runners=1",
				"--cluster-name=\"aks\"",
				"--namespace=\"namespace\"",
				"--pipeline=\"pipeline-value\"",
				"--id=\"processor-id\""},
			expected: "Processor [\"processor arBiTRary StrinG with 989784987  $#% 读写汉字 - 学中文 name 1\"] created\n",
		},
		"no_pipeline": {
			params: []string{
				"--name=\"Processor 1\"",
				"--sql=\"SET defaults.topic.autocreate=true; INSERT INTO topic-test-again-2 SELECT STREAM * FROM telecom_italia;\"",
				"--runners=1",
				"--cluster-name=\"aks\"",
				"--namespace=\"namespace\"",
				"--id=\"processor-id\""},
			expected: "Processor [\"Processor 1\"] created\n",
		},
		"no_id": {
			params: []string{
				"--name=\"Processor 2\"",
				"--sql=\"SET defaults.topic.autocreate=true; INSERT INTO topic-test-again-2 SELECT STREAM * FROM telecom_italia;\"",
				"--runners=1",
				"--cluster-name=\"aks\"",
				"--namespace=\"namespace\""},
			expected: "Processor [\"Processor 2\"] created\n",
		},
	}

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(string(processorRegisteredAsJSON)))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewProcessorCreateCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			output, err := test.ExecuteCommand(cmd, tc.params...)

			if !assert.Nil(t, err) {
				t.Fatalf(err.Error())
			}
			diff := cmp.Diff(tc.expected, output)
			if diff != "" {
				t.Fatalf(diff)
			}
		})
	}
	config.Client = nil
}

func TestNewProcessorCreateValidationErrors(t *testing.T) {
	tests := map[string]struct {
		params []string
	}{
		"all_params": {params: []string{
			"--name=\"processor arBiTRary StrinG with 989784987  $#% 读写汉字 - 学中文 name 1\"",
			"--sql=\"SET defaults.topic.autocreate=true; INSERT INTO topic-test-again-2 SELECT STREAM * FROM telecom_italia;\"",
			"--runners=1",
			"--cluster-name=\"aks\"",
			"--namespace=\"namespace\"",
			"--pipeline=\"pipeline-value\"",
			"--id=\"processor-id\""}},
		"no_pipeline": {params: []string{
			"--name=\"Processor 1\"",
			"--sql=\"SET defaults.topic.autocreate=true; INSERT INTO topic-test-again-2 SELECT STREAM * FROM telecom_italia;\"",
			"--runners=1",
			"--cluster-name=\"aks\"",
			"--namespace=\"namespace\"",
			"--id=\"processor-id\""}},
		"no_id": {params: []string{
			"--name=\"Processor 2\"",
			"--sql=\"SET defaults.topic.autocreate=true; INSERT INTO topic-test-again-2 SELECT STREAM * FROM telecom_italia;\"",
			"--runners=1",
			"--cluster-name=\"aks\"",
			"--namespace=\"namespace\""}},
	}

	var registerProcessorErrorResponse = "{ fields: [ {field: 'key-one', error: 'error-message'}], error: 'main-error-message'}"
	var registerProcessorErrorAsJSON, _ = json.Marshal(registerProcessorErrorResponse)

	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(string(registerProcessorErrorAsJSON)))
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := NewProcessorCreateCommand()
	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := test.ExecuteCommand(cmd, tc.params...)

			diff := cmp.Diff(string(registerProcessorErrorAsJSON), err.Error())
			if diff != "" {
				t.Fatalf(diff)
			}
		})
	}

	config.Client = nil
}
