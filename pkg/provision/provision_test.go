package provision

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/lensesio/lenses-go/v5/pkg/api"
	"gopkg.in/yaml.v2"
)

type MockHTTPClient struct{}

var bodyAsBytes []byte

func (m MockHTTPClient) Do(method, path, contentType string, send []byte, options ...api.RequestOption) (*http.Response, error) {

	body := FileUploadResp{
		"fdd0b728-7857-4da2-b016-00e51801719f",
		"nofile",
		1024,
		"admin",
	}

	bodyAsBytes, _ = json.Marshal(body)
	r := ioutil.NopCloser(bytes.NewReader(bodyAsBytes))
	return &http.Response{StatusCode: 200,
		Body: r}, nil
}

func (m MockHTTPClient) ReadResponseBody(resp *http.Response) ([]byte, error) {
	return bodyAsBytes, nil
}

const (
	rawURLInput = `
a: Easy!
b:
  fileRef:
    inline: "asdasdasda"
  d: [3, 4]
`
	rawURLOutput = `
a: Easy!
b:
  fileId: "fdd0b728-7857-4da2-b016-00e51801719f"
  d: [3, 4]
`

	emptyFileRefInput = `
a: Easy!
b:
  fileRef:
`
	invalidURLInput = `
a: Easy!
b:
  fileRef:
    URL: http_plain://acme.com
`
	nonExistanFileInput = `
a: Easy!
b:
  fileRef:
    filePath: /tmp/acme.txt
`

	validConfigInput = `
license:
  fileRef:
    URL: http://acme.com
connections:
  kafka:
    tags: []
    templateName: Kafka
    configurationObject:
      kafkaBootstrapServers:
        - SSL:///localhost:909093
      protocol: SSL
      sslKeystore:
        fileRef:
          inline: MIIFcwIBAzCCBSwGCSqGSIb3DQEHAaCCBR0EggUZMIIFFTCCBREGCSqGSIb3DQEHBqCCBQIwggT+AgEAMIIE9wYJKoZIhvcNAQcBMGYGCSqGSIb3DQEFDTBZMDgGCSqGSIb3DQEFDDArBBSZIsCyx6OD07VZBlkP1Wb1qGnasAICJxACASAwDAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEELIL4YHQocCrTZIlWZk7u4CAggSANkaRWh4bb5PQhDfeY+oMqUY7SVABU8fuyZYpTL3T9rOrmOlm9yq2lME/1CQYxlzr8pjv6tJfV2ciZNZde9eLfNTXtdRbFKZen0a1hfmvoSUHoEQReX7GQ7Cr/iACEW1nTZaZB9OiB3rF0h2eNtDlh+nMzr8QDwz8xye6sGSYKF2jhAn4IeAg/v+40/1Y+kjfK9MCOrLU593OqFZOJZGpm/oKE4pAsws+TiDMb25MiPeVgjLDJ4pY52fiIJLZ+t/jrNd/+8hfiAKpIRve4BtkMkfJMeOcRYUHoSdN+a6E//tPK0WdQIyoUEGZwmqCUnuLz9uXvnNFvBWb/ANduBudTPUaGUwOrMUBIvsZhDkh8uMw6aZ2L2Zy2IKDtvM10WERIDlY8qTa54JpxpeN/xMDSRpozLx07DETnUroFPHNCu7443gOTk21+6yLW3OlUQCf7kEbuT6yXOjD37ka27KAwf4ZfqPFAT3BmdtxAG/jwn58QbFIQHQ/Q4JwCIIUaG6Ke5GHcjm/Xip9RwAhF8SYtkFsxlBchez523hu4o83oTRLqh+pkvA4SoqkNvpAwrRMU/XYC55IeL8A99fxFsk2FN70iXksK8GuG54uInew8mMsIUEfX5IOwMAFQZ5NkVKMERCZcv1HrbjnrvSwLTa4st4VwOStZq67F437C9U1GobLXPyMMH0r+KnBHqgHQU+QLAhEmqt4SLR7TV3fGVdIqVC1wnBWQRoh+UmDcLGRB0R1Tv8a6vGhvQA7BUECtdRRbgMueRRL+Pkdh3wRTaVIvO24N5fTuAfsHKZKTRyvP0yaNOiJi+mkV/hueucCYJUXHDmV4kF2dlAiUXeSHF4jFx3aoYl9FjqUA7HMvOoK7i/CTDklK58qNE+Po7xPueX/scHRH626MB3l4gc7++Tug4XCQPVHzuVErWu1ybSb/rxFsD4fjyDh4hzab5iRjb+WyYORe6iFQiCaaDl3PpRtdckybl7N9wLsaU9FvhluJDVwbGYhYWNpnLv0WU3FE2bTODNLUW2H3vEuNNLtT8xsU/MEs/F7ScLUU+62wzAUCb09EO5jmocwcSXooGt0BUJqEEWK+GCr8dgwTmKWSUZ2VrOim9ag+wNXIsxrk31zlyd/f1JUQ6cIvnpYfAxKYk/jjBk26lXuVArZUS6Rmu/KzjEoplB450IQ2ihOJXdB7TxsnCquBNpM0z4pdfjXCBa/BoUjLBp90uLNoZbM8ZjaFb8Az8SL39jvZsMoplSONCRxZawgENiow1oWsLOg2vwad3u0E4erZ84pfyZkjDUTjDXDkuVcriMoeNgFqvnbXcS4qJmfJcFyfP0mpMLVigWbfNUx7A7jRvIj62Ma56nkshbpdHNN4f7Wsvas24fPwT80lU0dNMITqZXIRMkPYonN5wT/Spym19OBCuEb39IrOiaCFoLi2F9IW8QgKILMNXhQ8bNKH7i0+sql0NSDNl12x5aAkXwh+7ltP+HoXsV4TTFF1BhvViDVET55gbV8XhLSa6gh+A2UUu9l8OVKkoMqMD4wITAJBgUrDgMCGgUABBSpE4YA915s37bSch6n8X0QUkngAQQUX3TQXsAqVA7SPTvTb1HdfzxIyEECAwGGoA==
      sslKeyPassword: fastdata
      sslKeystorePassword: fastdata
      sslTruststore:
        fileRef:
          inline: MIIFcwIBAzCCBSwGCSqGSIb3DQEHAaCCBR0EggUZMIIFFTCCBREGCSqGSIb3DQEHBqCCBQIwggT+AgEAMIIE9wYJKoZIhvcNAQcBMGYGCSqGSIb3DQEFDTBZMDgGCSqGSIb3DQEFDDArBBSZIsCyx6OD07VZBlkP1Wb1qGnasAICJxACASAwDAYIKoZIhvcNAgkFADAdBglghkgBZQMEASoEELIL4YHQocCrTZIlWZk7u4CAggSANkaRWh4bb5PQhDfeY+oMqUY7SVABU8fuyZYpTL3T9rOrmOlm9yq2lME/1CQYxlzr8pjv6tJfV2ciZNZde9eLfNTXtdRbFKZen0a1hfmvoSUHoEQReX7GQ7Cr/iACEW1nTZaZB9OiB3rF0h2eNtDlh+nMzr8QDwz8xye6sGSYKF2jhAn4IeAg/v+40/1Y+kjfK9MCOrLU593OqFZOJZGpm/oKE4pAsws+TiDMb25MiPeVgjLDJ4pY52fiIJLZ+t/jrNd/+8hfiAKpIRve4BtkMkfJMeOcRYUHoSdN+a6E//tPK0WdQIyoUEGZwmqCUnuLz9uXvnNFvBWb/ANduBudTPUaGUwOrMUBIvsZhDkh8uMw6aZ2L2Zy2IKDtvM10WERIDlY8qTa54JpxpeN/xMDSRpozLx07DETnUroFPHNCu7443gOTk21+6yLW3OlUQCf7kEbuT6yXOjD37ka27KAwf4ZfqPFAT3BmdtxAG/jwn58QbFIQHQ/Q4JwCIIUaG6Ke5GHcjm/Xip9RwAhF8SYtkFsxlBchez523hu4o83oTRLqh+pkvA4SoqkNvpAwrRMU/XYC55IeL8A99fxFsk2FN70iXksK8GuG54uInew8mMsIUEfX5IOwMAFQZ5NkVKMERCZcv1HrbjnrvSwLTa4st4VwOStZq67F437C9U1GobLXPyMMH0r+KnBHqgHQU+QLAhEmqt4SLR7TV3fGVdIqVC1wnBWQRoh+UmDcLGRB0R1Tv8a6vGhvQA7BUECtdRRbgMueRRL+Pkdh3wRTaVIvO24N5fTuAfsHKZKTRyvP0yaNOiJi+mkV/hueucCYJUXHDmV4kF2dlAiUXeSHF4jFx3aoYl9FjqUA7HMvOoK7i/CTDklK58qNE+Po7xPueX/scHRH626MB3l4gc7++Tug4XCQPVHzuVErWu1ybSb/rxFsD4fjyDh4hzab5iRjb+WyYORe6iFQiCaaDl3PpRtdckybl7N9wLsaU9FvhluJDVwbGYhYWNpnLv0WU3FE2bTODNLUW2H3vEuNNLtT8xsU/MEs/F7ScLUU+62wzAUCb09EO5jmocwcSXooGt0BUJqEEWK+GCr8dgwTmKWSUZ2VrOim9ag+wNXIsxrk31zlyd/f1JUQ6cIvnpYfAxKYk/jjBk26lXuVArZUS6Rmu/KzjEoplB450IQ2ihOJXdB7TxsnCquBNpM0z4pdfjXCBa/BoUjLBp90uLNoZbM8ZjaFb8Az8SL39jvZsMoplSONCRxZawgENiow1oWsLOg2vwad3u0E4erZ84pfyZkjDUTjDXDkuVcriMoeNgFqvnbXcS4qJmfJcFyfP0mpMLVigWbfNUx7A7jRvIj62Ma56nkshbpdHNN4f7Wsvas24fPwT80lU0dNMITqZXIRMkPYonN5wT/Spym19OBCuEb39IrOiaCFoLi2F9IW8QgKILMNXhQ8bNKH7i0+sql0NSDNl12x5aAkXwh+7ltP+HoXsV4TTFF1BhvViDVET55gbV8XhLSa6gh+A2UUu9l8OVKkoMqMD4wITAJBgUrDgMCGgUABBSpE4YA915s37bSch6n8X0QUkngAQQUX3TQXsAqVA7SPTvTb1HdfzxIyEECAwGGoA==
      sslTruststorePassword: fastdata
  schema-registry:
    templateName: SchemaRegiststries
    tags: []
    configurationObject:
      schemaRegistryUrls:
        - http://0.0.0.0:80811
      metricsPort: 9582
      metricsType: JMX
      metricsSsl: false
      additionalProperties: {}
`
	validConfigOutput = `
license:
  fileRef:
    URL: http://acme.com
connections:
  kafka:
    tags: []
    templateName: Kafka
    configurationObject:
      kafkaBootstrapServers:
        - SSL:///localhost:909093
      protocol: SSL
      sslKeystore:
        fileId: "fdd0b728-7857-4da2-b016-00e51801719f"
      sslKeyPassword: fastdata
      sslKeystorePassword: fastdata
      sslTruststore:
        fileId: "fdd0b728-7857-4da2-b016-00e51801719f"
      sslTruststorePassword: fastdata
  schema-registry:
    templateName: SchemaRegiststries
    tags: []
    configurationObject:
      schemaRegistryUrls:
        - http://0.0.0.0:80811
      metricsPort: 9582
      metricsType: JMX
      metricsSsl: false
      additionalProperties: {}
`
)

func Test_parseConfig(t *testing.T) {
	// Mock HTTP client that always return '200' and a valid file upload response body
	client := MockHTTPClient{}

	type args struct {
		input string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name: "first of its kind",
			args: args{
				rawURLInput,
			},
			wantErr: false,
		},
		{
			name: "empty fileRef key",
			args: args{
				emptyFileRefInput,
			},
			wantErr: true,
		},
		{
			name: "invalid URL value",
			args: args{
				invalidURLInput,
			},
			wantErr: true,
		},
		{
			name: "non existant filepath reference",
			args: args{
				nonExistanFileInput,
			},
			wantErr: true,
		},
		{
			name: "valid partial config",
			args: args{
				validConfigInput,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var inputMap map[interface{}]interface{}

			err := yaml.Unmarshal([]byte(tt.args.input), &inputMap)
			if err != nil {
				t.Fatalf("%s", err)
			}

			err = parseConfig(inputMap, client)
			if (err != nil) != tt.wantErr {
				t.Errorf("parseConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

const (
	noLicenseConfigInput = `
connections:
  kafka:
    tags: []
    templateName: Kafka
    configurationObject:
      kafkaBootstrapServers:
        - PLAINTEXT:///localhost:9093
      protocol: PLAINTEXT
      additionalProperties:
        proprtyOne: "1"
        proprtyTwo: "2"
`
	noConnectionsConfigInput = `
license:
  fileRef:
    URL: http://acme.com
`

	invalidConnectionsStructureConfigInput = `
license:
  fileRef:
    URL: http://acme.com

connections:
  - name: kafka
    tags: []
    templateName: Kafka
    configurationObject:
      kafkaBootstrapServers:
        - PLAINTEXT:///localhost:9092
      protocol: PLAINTEXT
`
)

func Test_checkConfigValidity(t *testing.T) {
	type args struct {
		config string
	}
	tests := []struct {
		name string
		args args
		err  error
	}{
		{
			name: "valid input",
			args: args{
				config: validConfigInput,
			},
			err: nil,
		},
		{
			name: "valid output",
			args: args{
				config: validConfigOutput,
			},
			err: nil,
		},
		{
			name: "no connections",
			args: args{
				config: noConnectionsConfigInput,
			},
			err: errMissingConnections,
		},
		{
			name: "invalid connections structure",
			args: args{
				config: invalidConnectionsStructureConfigInput,
			},
			err: errInvalidConnectionsStruct,
		},
	}

	for _, tt := range tests {
		inputMap := make(map[interface{}]interface{})

		t.Run(tt.name, func(t *testing.T) {
			if uerr := yaml.Unmarshal([]byte(tt.args.config), &inputMap); uerr != nil {
				t.Fatalf("failed to unmarshall config input, error = %v", uerr)
			}

			if err := checkConfigValidity(inputMap); err != tt.err {
				t.Errorf("checkConfigValidity() error = %v, wanted error %v", err, tt.err)
			}
		})
	}
}
