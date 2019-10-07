package secret

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	keyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	azure "github.com/Azure/go-autorest/autorest/azure"
	azureAuth "github.com/Azure/go-autorest/autorest/azure/auth"
	vaultapi "github.com/hashicorp/vault/api"
	golog "github.com/kataras/golog"
	"github.com/landoop/lenses-go/pkg/utils"
)

var providerType Provider

const (
	// SecretPreFix for looking up env vars
	SecretPreFix = "SECRET_"
	// ConnectWorkerSecretPrefix prefix for connect worker secrets
	ConnectWorkerSecretPrefix = "WORKER_CONNECT_SECRET_"
	// EnvVaultRole is the environment var holding the vault role
	EnvVaultRole = "VAULT_ROLE"
	// EnvAzureClientID is the environment var holding the azure client id
	EnvAzureClientID = "AZURE_CLIENT_ID"
	// EnvAzureClientSecret is the environment var holding the azure client secret
	EnvAzureClientSecret = "AZURE_CLIENT_SECRET"
	// EnvAzureClientTenantID is the environment var holding the azure tenant id
	EnvAzureClientTenantID = "AZURE_TENANT_ID"
	// EnvAzureKeyVaultName is the environment var holding the azure key vault name
	EnvAzureKeyVaultName = "AZURE_KEY_VAULT"
	// EnvAzureKeyVaultDNS is the environment var holding the azure key dns
	EnvAzureKeyVaultDNS = "AZURE_KEY_VAULT_DNS"
)

// Provider secret provider type
type Provider string

const (
	// Vault Hashicorp Vault
	Vault Provider = "VAULT"
	// AzureKV Azure KeyVault
	AzureKV Provider = "AZURE_KV"
	// Kubernetes Kubernetes secrets
	Kubernetes Provider = "KUBERNETES"
)

type vaultAppRoleCredentials struct {
	RoleID   string
	SecretID string
}

// Secret holds the mapping of the env var to the secret key
type Secret struct {
	EnvKey string
	Key    string
	Value  string
}

func loadSecrets(file string) ([]string, error) {
	var secretVars []string

	if file != "" {
		golog.Infof("Loading variables from file [%s]", file)
		lines, err := utils.ReadLines(file)

		for _, l := range lines {
			if strings.HasPrefix(l, SecretPreFix) || strings.HasPrefix(l, ConnectWorkerSecretPrefix) {
				secretVars = append(secretVars, strings.Replace(l, SecretPreFix, "", -1))
			}
		}

		if err != nil {
			return secretVars, err
		}

	} else {
		secretVars = getSecretVars()
	}

	if len(secretVars) == 0 {
		golog.Warnf("No environment variables prefixed with [%s] found or loaded from file [%s]", SecretPreFix, file)
	}

	return secretVars, nil
}

func getSecretVars() []string {
	var secretVars []string
	// get secret vars
	golog.Info("Looking for secret environment variables")
	vars := os.Environ()

	for _, v := range vars {
		if strings.HasPrefix(v, SecretPreFix) || strings.HasPrefix(v, ConnectWorkerSecretPrefix) {
			golog.Infof("Found environment var [%s]", v)
			secretVars = append(secretVars, v)
		}
	}

	return secretVars
}

func getVaultClient(server, token string) (*vaultapi.Client, error) {

	config := vaultapi.Config{}

	// read the config from the environment
	config.ReadEnvironment()

	// set the address is we have nothing
	if config.Address == "" {
		if server == "" {
			golog.Error(`Required flag "vault-addr" not set and no VAULT_ADDR environment variable found`)
			return nil, errors.New(``)
		}
		config.Address = server
	}

	client, err := vaultapi.NewClient(&config)

	if err != nil {
		golog.Errorf("Failed to create vault client for server [%s]", server)
		return nil, err
	}

	// set the provided token if none is present in the environment
	if envToken := os.Getenv(vaultapi.EnvVaultToken); envToken == "" {
		if token == "" {
			golog.Error(`Required flag "vault-token" not set and no VAULT_TOKEN environment variable found`)
			return nil, errors.New(``)
		}
		client.SetToken(token)
	}

	return client, nil
}

func getVaultAppIDs(client *vaultapi.Client, role string) (vaultAppRoleCredentials, error) {
	var ids = vaultAppRoleCredentials{}

	roleID, err := client.Logical().Read(fmt.Sprintf("auth/approle/role/%s/role-id", role))

	if err != nil {
		golog.Errorf("Failed to read role-id for role [%s] on server [%s]", role, client.Address())
		return ids, err
	}

	secretID, err := client.Logical().Write(fmt.Sprintf("auth/approle/role/%s/secret-id", role), map[string]interface{}{})

	if err != nil {
		golog.Errorf("Failed to read secret-id for role [%s] on server [%s]", role, client.Address())
		return ids, err
	}

	ids.RoleID = roleID.Data["role_id"].(string)
	ids.SecretID = secretID.Data["secret_id"].(string)

	return ids, nil
}

func vaultAppRoleLogin(client *vaultapi.Client, credentials vaultAppRoleCredentials) error {

	data := map[string]interface{}{
		"role_id":   credentials.RoleID,
		"secret_id": credentials.SecretID,
	}

	secret, err := client.Logical().Write("auth/approle/login", data)

	if err != nil {
		golog.Error("Failed to login as appRole")
		return err
	}

	token := secret.Auth.ClientToken

	client.ClearToken()
	client.SetToken(token)

	return nil
}

// VaultConnectExternalHandler retrieves secret key values from Vault based on environment variables
func VaultConnectExternalHandler(role, token, endpoint, file string) ([]Secret, error) {
	var secrets []Secret
	var secretVars []string

	client, err := getVaultClient(endpoint, token)

	if err != nil {
		return nil, err
	}

	if envRole := os.Getenv(EnvVaultRole); envRole != "" {
		role = envRole
	} else {
		if role == "" {
			golog.Error(`Required flag "vault-role" not set and no VAULT_ROLE environment variable found`)
			return nil, errors.New("")
		}
	}

	ids, err := getVaultAppIDs(client, role)

	// load secrets from environment or file
	secretVars, varErr := loadSecrets(file)

	if varErr != nil {
		return nil, varErr
	}

	if err != nil {
		return nil, err
	}

	// login as appRole
	vaultAppRoleLogin(client, ids)

	server := endpoint
	if endpoint == "" {
		server = os.Getenv(vaultapi.EnvVaultAddress)
	}
	logical := client.Logical()

	for _, v := range secretVars {
		split := strings.SplitN(v, "=", 2)
		path := split[1]
		envKey := split[0]
		key := strings.Replace(strings.Replace(envKey, ConnectWorkerSecretPrefix, "", 1), SecretPreFix, "", 1)
		keyForVault := strings.ToLower(strings.Replace(key, "_", "-", -1))
		keyAsProp := strings.ToLower(strings.Replace(key, "_", ".", -1))

		golog.Infof(fmt.Sprintf("Retrieving secret from server [%s]. EnvVar: [%s], Path: [%s], Key: [%s]", server, v, path, keyForVault))
		secret, err := logical.Read(fmt.Sprintf("%s", path))

		if err != nil {
			golog.Errorf("Failed to retrieve secret for path [%s] and key [%s] from server [%s]. Error [%s]", path, keyForVault, endpoint, err.Error())
			continue
		}

		if secret == nil {
			golog.Errorf("Failed to retrieve secret for path [%s] from server [%s]. No secret data return. Possible bad path", path, endpoint)
			continue
		}

		if data, ok := secret.Data["data"]; ok && data != nil {
			var val interface{}
			switch data.(type) {
			case map[string]interface{}:
				val = data.(map[string]interface{})[keyForVault]
			}

			secret := fmt.Sprintf("%v", val)
			if secret != "" {
				golog.Infof("Found secret for EnvVar: [%s], Key: [%s] in endpoint [%s]", key, keyForVault, endpoint)
				secrets = append(secrets, Secret{EnvKey: envKey, Key: keyAsProp, Value: secret})
			}
		}

	}

	return secrets, nil
}

// AzureConfiguration holds azure configuration details
type AzureConfiguration struct {
	ClientID     string
	ClientSecret string
	TenantID     string
	KeyVaultName string
}

// KeyVault holds the information for a keyvault instance
type KeyVault struct {
	client   *keyvault.BaseClient
	vaultURL string
}

// AzureKeyVaultHandler retrieves secret key values from Azure KeyVault based on environment variables
func AzureKeyVaultHandler(vaultURL, file string, config AzureConfiguration) ([]Secret, error) {
	var secretVars []string
	var secrets []Secret

	// load secrets from environment or file
	secretVars, err := loadSecrets(file)

	if err != nil {
		return nil, err
	}

	// get a vault client
	client, err := newKeyVaultClient(vaultURL, config)

	if err != nil {
		golog.Errorf("Failed to get Azure key client for vault [%s]. [%s]", vaultURL, err.Error())
		return nil, err
	}

	for _, v := range secretVars {
		split := strings.SplitN(v, "=", 2)
		envKey := split[0]
		key := strings.Replace(strings.Replace(envKey, ConnectWorkerSecretPrefix, "", 1), SecretPreFix, "", 1)

		keyForAz := strings.ToLower(strings.Replace(key, "_", "-", -1))
		keyAsProp := strings.ToLower(strings.Replace(key, "_", ".", -1))
		golog.Infof(fmt.Sprintf("Retrieving secret from vault [%s]. EnvVar: [%s], Key: [%s]", vaultURL, v, keyForAz))
		secret, err := client.getSecret(keyForAz)

		if err != nil {
			golog.Errorf("Failed to retrieve secret for key [%s] from vault [%s]. Error [%s]", keyForAz, vaultURL, err.Error())
			continue
		}

		if secret != "" {
			golog.Infof("Found secret for EnvVar: [%s], Key: [%s] in vault [%s]", key, keyForAz, vaultURL)
			secrets = append(secrets, Secret{EnvKey: envKey, Key: keyAsProp, Value: secret})
		}
	}

	return secrets, nil
}

// NewKeyVaultClient creates a new keyvault client
func newKeyVaultClient(vaultURL string, config AzureConfiguration) (*KeyVault, error) {

	keyClient := keyvault.New()

	credentials := azureAuth.NewClientCredentialsConfig(config.ClientID, config.ClientSecret, config.TenantID)
	// set the correct resource for keyvault, trim the trailing /
	credentials.Resource = strings.TrimRight(azure.PublicCloud.KeyVaultEndpoint, "/")
	a, err := credentials.Authorizer()

	if err != nil {
		golog.Errorf("Failed to create KeyVault client. [%s]", err.Error())
		return nil, err
	}

	keyClient.Authorizer = a

	k := &KeyVault{
		vaultURL: vaultURL,
		client:   &keyClient,
	}

	return k, nil
}

// GetSecret retrieves a secret from keyvault
func (k *KeyVault) getSecret(keyName string) (string, error) {
	ctx := context.Background()

	keyBundle, err := k.client.GetSecret(ctx, k.vaultURL, keyName, "")
	if err != nil {
		return "", err
	}

	return *keyBundle.Value, nil
}

// EnvSecretHandler retrieves secret key values from environment variables
func EnvSecretHandler(file string) ([]Secret, error) {
	var secretVars []string
	var secrets []Secret

	// load secrets from environment or file
	secretVars, err := loadSecrets(file)

	if err != nil {
		return nil, err
	}

	for _, v := range secretVars {
		split := strings.SplitN(v, "=", 2)
		envKey := split[0]
		key := strings.Replace(strings.Replace(envKey, ConnectWorkerSecretPrefix, "", 1), SecretPreFix, "", 1)
		value := split[1]
		keyAsProp := strings.ToLower(strings.Replace(key, "_", ".", -1))
		secrets = append(secrets, Secret{EnvKey: envKey, Key: keyAsProp, Value: value})
	}

	return secrets, nil
}
