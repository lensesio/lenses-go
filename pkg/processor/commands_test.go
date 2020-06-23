package processor

import (
	"encoding/json"
	"net/http"
	"testing"
	"strings"

	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	test "github.com/landoop/lenses-go/test"
	"github.com/stretchr/testify/assert"
)


var aksk8s = api.KubernetesTarget{Cluster: "aks", Namespaces: []string{"prod"}, Version: "1.0.0" }
var aksk8Res = ListTargetsResult{Type: "Kubernetes", ClusterName: "aks", Namespace: "prod", Version: "1.0.0" }
var eksk8s = api.KubernetesTarget{Cluster: "eks", Namespaces: []string{"dev"}, Version: "1.0.0" }
var eksk8Res = ListTargetsResult{Type: "Kubernetes", ClusterName: "eks", Namespace: "dev", Version: "1.0.0" }
var targetList = &api.DeploymentTargets{
	Kubernetes: []api.KubernetesTarget{aksk8s, eksk8s},
}

var targetsAsJSON, _ = json.Marshal(targetList)


func TestListTargetDeploymentCommand(t *testing.T) {

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