package elasticsearch

import (
	"net/http"
	"testing"

	"github.com/landoop/lenses-go/pkg/api"
	config "github.com/landoop/lenses-go/pkg/configs"
	"github.com/landoop/lenses-go/test"
	"github.com/stretchr/testify/assert"
)

const indexesOkResponse = `
[{"indexName":"dev-complex","connectionName":"es6again","keyType":"STRING","valueType":"JSON","keySchema":"\"string\"","valueSchema":"{\"type\":\"record\",\"name\":\"lenses_record\",\"namespace\":\"lenses\",\"fields\":[{\"name\":\"Customer\",\"type\":{\"type\":\"record\",\"name\":\"Customer\",\"fields\":[{\"name\":\"contactInfo\",\"type\":{\"type\":\"record\",\"name\":\"contactInfo\",\"fields\":[{\"name\":\"phone1\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"coordinates\",\"type\":{\"type\":\"record\",\"name\":\"coordinates\",\"fields\":[{\"name\":\"lat\",\"type\":\"float\",\"doc\":\"float\"},{\"name\":\"lng\",\"type\":\"float\",\"doc\":\"float\"}]}},{\"name\":\"email2\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"city\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"state\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"email1\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"address2\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"postalCode\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"address1\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"phone2\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"personalInfo\",\"type\":{\"type\":\"record\",\"name\":\"personalInfo\",\"fields\":[{\"name\":\"bankAccount\",\"type\":{\"type\":\"record\",\"name\":\"bankAccount\",\"fields\":[{\"name\":\"bankISOCode\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"branchIdentifier\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"bankIdentifier\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"accountNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"ibanCheckDigits\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"iban\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"bban\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"medicalInfo\",\"type\":{\"type\":\"record\",\"name\":\"medicalInfo\",\"fields\":[{\"name\":\"bloodType\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"notes\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"legalInfo\",\"type\":{\"type\":\"record\",\"name\":\"legalInfo\",\"fields\":[{\"name\":\"military\",\"type\":{\"type\":\"record\",\"name\":\"military\",\"fields\":[{\"name\":\"hasServed\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"info\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"passport\",\"type\":{\"type\":\"record\",\"name\":\"passport\",\"fields\":[{\"name\":\"expiry\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"nationality\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"passportID\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"ssn\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"workInfo\",\"type\":{\"type\":\"record\",\"name\":\"workInfo\",\"fields\":[{\"name\":\"legalHQ\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"phoneNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"ein\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"job\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"company\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"conpanyCID\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"creditCards\",\"type\":{\"type\":\"record\",\"name\":\"creditCards\",\"fields\":[{\"name\":\"card1\",\"type\":{\"type\":\"record\",\"name\":\"card1\",\"fields\":[{\"name\":\"ccNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"expiry\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"provider\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"card2\",\"type\":{\"type\":\"record\",\"name\":\"card2\",\"fields\":[{\"name\":\"ccNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"expiry\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"provider\",\"type\":\"string\",\"doc\":\"string\"}]}}]}}]}},{\"name\":\"lastName\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"firstName\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"birthDate\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"gender\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"transactionsPortal\",\"type\":{\"type\":\"record\",\"name\":\"transactionsPortal\",\"fields\":[{\"name\":\"connectionInfo\",\"type\":{\"type\":\"record\",\"name\":\"connectionInfo\",\"fields\":[{\"name\":\"ipv4\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"lastLogin\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"macAddress\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"agent\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"usageInfo\",\"type\":{\"type\":\"record\",\"name\":\"usageInfo\",\"fields\":[{\"name\":\"idleTime\",\"type\":\"long\",\"doc\":\"long\"},{\"name\":\"mostVisitedPage\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"ipv6\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"transaction\",\"type\":{\"type\":\"record\",\"name\":\"transaction\",\"fields\":[{\"name\":\"ID\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"timestamp\",\"type\":\"float\",\"doc\":\"float\"},{\"name\":\"amount\",\"type\":\"long\",\"doc\":\"long\"},{\"name\":\"currency\",\"type\":{\"type\":\"record\",\"name\":\"currency\",\"fields\":[{\"name\":\"code\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"name\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"EAN\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"username\",\"type\":\"string\",\"doc\":\"string\"}]}}]}","size":4244452,"totalMessages":1001,"status":"yellow","shardsCount":5,"replicas":5,"permissions":["ShowTopic","QueryTopic","ViewSchema"]},{"indexName":"data-complex","connectionName":"es6again","keyType":"STRING","valueType":"JSON","keySchema":"\"string\"","valueSchema":"{\"type\":\"record\",\"name\":\"lenses_record\",\"namespace\":\"lenses\",\"fields\":[{\"name\":\"Customer\",\"type\":{\"type\":\"record\",\"name\":\"Customer\",\"fields\":[{\"name\":\"contactInfo\",\"type\":{\"type\":\"record\",\"name\":\"contactInfo\",\"fields\":[{\"name\":\"phone1\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"coordinates\",\"type\":{\"type\":\"record\",\"name\":\"coordinates\",\"fields\":[{\"name\":\"lat\",\"type\":\"float\",\"doc\":\"float\"},{\"name\":\"lng\",\"type\":\"float\",\"doc\":\"float\"}]}},{\"name\":\"email2\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"city\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"state\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"email1\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"address2\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"postalCode\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"address1\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"phone2\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"personalInfo\",\"type\":{\"type\":\"record\",\"name\":\"personalInfo\",\"fields\":[{\"name\":\"bankAccount\",\"type\":{\"type\":\"record\",\"name\":\"bankAccount\",\"fields\":[{\"name\":\"bankISOCode\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"branchIdentifier\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"bankIdentifier\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"accountNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"ibanCheckDigits\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"iban\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"bban\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"medicalInfo\",\"type\":{\"type\":\"record\",\"name\":\"medicalInfo\",\"fields\":[{\"name\":\"bloodType\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"notes\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"legalInfo\",\"type\":{\"type\":\"record\",\"name\":\"legalInfo\",\"fields\":[{\"name\":\"military\",\"type\":{\"type\":\"record\",\"name\":\"military\",\"fields\":[{\"name\":\"hasServed\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"info\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"passport\",\"type\":{\"type\":\"record\",\"name\":\"passport\",\"fields\":[{\"name\":\"expiry\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"nationality\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"passportID\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"ssn\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"workInfo\",\"type\":{\"type\":\"record\",\"name\":\"workInfo\",\"fields\":[{\"name\":\"legalHQ\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"phoneNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"ein\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"job\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"company\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"conpanyCID\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"creditCards\",\"type\":{\"type\":\"record\",\"name\":\"creditCards\",\"fields\":[{\"name\":\"card1\",\"type\":{\"type\":\"record\",\"name\":\"card1\",\"fields\":[{\"name\":\"ccNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"expiry\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"provider\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"card2\",\"type\":{\"type\":\"record\",\"name\":\"card2\",\"fields\":[{\"name\":\"ccNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"expiry\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"provider\",\"type\":\"string\",\"doc\":\"string\"}]}}]}}]}},{\"name\":\"lastName\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"firstName\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"birthDate\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"gender\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"transactionsPortal\",\"type\":{\"type\":\"record\",\"name\":\"transactionsPortal\",\"fields\":[{\"name\":\"connectionInfo\",\"type\":{\"type\":\"record\",\"name\":\"connectionInfo\",\"fields\":[{\"name\":\"ipv4\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"lastLogin\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"macAddress\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"agent\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"usageInfo\",\"type\":{\"type\":\"record\",\"name\":\"usageInfo\",\"fields\":[{\"name\":\"idleTime\",\"type\":\"long\",\"doc\":\"long\"},{\"name\":\"mostVisitedPage\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"ipv6\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"transaction\",\"type\":{\"type\":\"record\",\"name\":\"transaction\",\"fields\":[{\"name\":\"ID\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"timestamp\",\"type\":\"float\",\"doc\":\"float\"},{\"name\":\"amount\",\"type\":\"long\",\"doc\":\"long\"},{\"name\":\"currency\",\"type\":{\"type\":\"record\",\"name\":\"currency\",\"fields\":[{\"name\":\"code\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"name\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"EAN\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"username\",\"type\":\"string\",\"doc\":\"string\"}]}}]}","size":4141656,"totalMessages":1001,"status":"yellow","shardsCount":5,"replicas":5,"permissions":["ShowTopic","QueryTopic","ViewSchema"]}]
`

const indexOkResponse = `
	{"indexName":"dev-complex","connectionName":"es6again","keyType":"STRING","valueType":"JSON","keySchema":"\"string\"","valueSchema":"{\"type\":\"record\",\"name\":\"lenses_record\",\"namespace\":\"lenses\",\"fields\":[{\"name\":\"Customer\",\"type\":{\"type\":\"record\",\"name\":\"Customer\",\"fields\":[{\"name\":\"contactInfo\",\"type\":{\"type\":\"record\",\"name\":\"contactInfo\",\"fields\":[{\"name\":\"phone1\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"coordinates\",\"type\":{\"type\":\"record\",\"name\":\"coordinates\",\"fields\":[{\"name\":\"lat\",\"type\":\"float\",\"doc\":\"float\"},{\"name\":\"lng\",\"type\":\"float\",\"doc\":\"float\"}]}},{\"name\":\"email2\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"city\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"state\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"email1\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"address2\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"postalCode\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"address1\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"phone2\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"personalInfo\",\"type\":{\"type\":\"record\",\"name\":\"personalInfo\",\"fields\":[{\"name\":\"bankAccount\",\"type\":{\"type\":\"record\",\"name\":\"bankAccount\",\"fields\":[{\"name\":\"bankISOCode\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"branchIdentifier\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"bankIdentifier\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"accountNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"ibanCheckDigits\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"iban\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"bban\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"medicalInfo\",\"type\":{\"type\":\"record\",\"name\":\"medicalInfo\",\"fields\":[{\"name\":\"bloodType\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"notes\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"legalInfo\",\"type\":{\"type\":\"record\",\"name\":\"legalInfo\",\"fields\":[{\"name\":\"military\",\"type\":{\"type\":\"record\",\"name\":\"military\",\"fields\":[{\"name\":\"hasServed\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"info\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"passport\",\"type\":{\"type\":\"record\",\"name\":\"passport\",\"fields\":[{\"name\":\"expiry\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"nationality\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"passportID\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"ssn\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"workInfo\",\"type\":{\"type\":\"record\",\"name\":\"workInfo\",\"fields\":[{\"name\":\"legalHQ\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"phoneNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"ein\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"job\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"company\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"conpanyCID\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"creditCards\",\"type\":{\"type\":\"record\",\"name\":\"creditCards\",\"fields\":[{\"name\":\"card1\",\"type\":{\"type\":\"record\",\"name\":\"card1\",\"fields\":[{\"name\":\"ccNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"expiry\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"provider\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"card2\",\"type\":{\"type\":\"record\",\"name\":\"card2\",\"fields\":[{\"name\":\"ccNumber\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"expiry\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"provider\",\"type\":\"string\",\"doc\":\"string\"}]}}]}}]}},{\"name\":\"lastName\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"firstName\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"birthDate\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"gender\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"transactionsPortal\",\"type\":{\"type\":\"record\",\"name\":\"transactionsPortal\",\"fields\":[{\"name\":\"connectionInfo\",\"type\":{\"type\":\"record\",\"name\":\"connectionInfo\",\"fields\":[{\"name\":\"ipv4\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"lastLogin\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"macAddress\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"agent\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"usageInfo\",\"type\":{\"type\":\"record\",\"name\":\"usageInfo\",\"fields\":[{\"name\":\"idleTime\",\"type\":\"long\",\"doc\":\"long\"},{\"name\":\"mostVisitedPage\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"ipv6\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"transaction\",\"type\":{\"type\":\"record\",\"name\":\"transaction\",\"fields\":[{\"name\":\"ID\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"timestamp\",\"type\":\"float\",\"doc\":\"float\"},{\"name\":\"amount\",\"type\":\"long\",\"doc\":\"long\"},{\"name\":\"currency\",\"type\":{\"type\":\"record\",\"name\":\"currency\",\"fields\":[{\"name\":\"code\",\"type\":\"string\",\"doc\":\"string\"},{\"name\":\"name\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"EAN\",\"type\":\"string\",\"doc\":\"string\"}]}},{\"name\":\"username\",\"type\":\"string\",\"doc\":\"string\"}]}}]}","size":4244452,"totalMessages":1001,"status":"yellow","shards":[{"shard":"0","records":197,"replicas":1,"availableReplicas":0},{"shard":"1","records":191,"replicas":1,"availableReplicas":0},{"shard":"2","records":211,"replicas":1,"availableReplicas":0},{"shard":"3","records":201,"replicas":1,"availableReplicas":0},{"shard":"4","records":201,"replicas":1,"availableReplicas":0}],"replicas":5,"permissions":["ShowTopic","QueryTopic","ViewSchema"]}
`

func TestIndexesCommandSuccess(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(indexesOkResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	var outputValue string
	cmd := IndexesCommand()

	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	output, err := test.ExecuteCommand(cmd)

	assert.Nil(t, err)
	assert.NotEmpty(t, output)
	config.Client = nil
}

func TestIndexesCommandFail(t *testing.T) {
	//setup http client
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	_, indexErr := test.ExecuteCommand(IndexesCommand())

	assert.NotNil(t, indexErr)
	config.Client = nil
}

func TestIndexCommenad(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(indexOkResponse))
	})

	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, _ := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	config.Client = client

	cmd := IndexCommand()

	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	output, err := test.ExecuteCommand(cmd, `--connection="lorem"`, `--name="ipsum"`)

	assert.NotEmpty(t, output)
	assert.Nil(t, err)

	config.Client = nil
}

func TestIndexCommendFail(t *testing.T) {
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})
	httpClient, teardown := test.TestingHTTPClient(h)
	defer teardown()

	client, err := api.OpenConnection(test.ClientConfig, api.UsingClient(httpClient))

	assert.Nil(t, err)

	config.Client = client

	cmd := IndexCommand()

	var outputValue string
	cmd.PersistentFlags().StringVar(&outputValue, "output", "json", "")

	_, indexErr := test.ExecuteCommand(cmd, `--connection="lorem"`, `--name="ipsum"`)

	assert.NotNil(t, indexErr)
	config.Client = nil
}
