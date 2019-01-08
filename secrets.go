package lenses

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	keyvault "github.com/Azure/azure-sdk-for-go/services/keyvault/2016-10-01/keyvault"
	azureAuth "github.com/Azure/go-autorest/autorest/azure/auth"
	vaultapi "github.com/hashicorp/vault/api"
	golog "github.com/kataras/golog"
)

var providerType SecretProvider

// SecretProvider secret provider type
type SecretProvider string

const (
	// Vault Hashicorp Vault
	Vault SecretProvider = "VAULT"
	// AzureKV Azure KeyVault
	AzureKV SecretProvider = "AZURE_KV"
	// Kubernetes Kubernetes secrets
	Kubernetes SecretProvider = "KUBERNETES"
)

type vaultAppRoleCredentials struct {
	RoleID   string
	SecretID string
}

func loadSecrets(file string) ([]string, error) {
	var secretVars []string

	if file != "" {
		golog.Infof("Loading variables from file [%s]", file)
		lines, err := ReadLines(file)

		for _, l := range lines {
			if strings.HasPrefix(l, "SECRET_") {
				secretVars = append(secretVars, strings.Replace(l, "SECRET_", "", -1))
			}
		}

		if err != nil {
			return secretVars, err
		}

	} else {
		secretVars = getSecretVars()
	}

	if len(secretVars) == 0 {
		golog.Warnf("No environment variables prefixed with [SECRET_] found or loaded from file [%s]", file)
	}

	return secretVars, nil
}

func getSecretVars() []string {
	var secretVars []string
	// get secret vars
	golog.Info("Looking for secret environment variables")
	vars := os.Environ()

	for _, v := range vars {
		if strings.HasPrefix(v, "SECRET_") {
			golog.Infof("Found environment var [%s]", v)
			secretVars = append(secretVars, strings.Replace(v, "SECRET_", "", -1))
		}
	}

	return secretVars
}

func getVaultClient(server, token string) (*vaultapi.Client, error) {
	client, err := vaultapi.NewClient(&vaultapi.Config{
		Address: server,
	})

	if err != nil {
		golog.Errorf("Failed to create vault client for server [%s]", server)
		return nil, err
	}

	client.SetToken(token)

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
func VaultConnectExternalHandler(role, token, endpoint, file string) (map[string]string, error) {
	secrets := make(map[string]string)
	var secretVars []string

	// load secrets from environment or file
	secretVars, err := loadSecrets(file)

	if err != nil {
		return nil, err
	}

	client, err := getVaultClient(endpoint, token)

	if err != nil {
		return nil, err
	}

	ids, err := getVaultAppIDs(client, role)

	if err != nil {
		return nil, err
	}

	// login as appRole
	vaultAppRoleLogin(client, ids)

	logical := client.Logical()

	for _, v := range secretVars {
		split := strings.Split(v, "=")
		path := split[1]
		key := split[0]
		keyForVault := strings.ToLower(strings.Replace(key, "_", "-", -1))
		keyAsProp := strings.ToLower(strings.Replace(key, "_", ".", -1))

		golog.Infof(fmt.Sprintf("Retrieving secret from server [%s]. EnvVar: [%s], Path: [%s], Key: [%s]", endpoint, v, path, keyForVault))
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
				secrets[keyAsProp] = secret
			}
			secrets[keyAsProp] = fmt.Sprintf("%v", val)
		}

	}

	return secrets, nil
}

// AzureConfiguration holds azure configuration details
type AzureConfiguration struct {
	ClientID     string
	ClientSecret string
	TenantID     string
}

// KeyVault holds the information for a keyvault instance
type KeyVault struct {
	client   *keyvault.BaseClient
	vaultURL string
}

// ReadLines reads a file and returns the string contents
func ReadLines(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}

// AzureKeyVaultHandler retrieves secret key values from Azure KeyVault based on environment variables
func AzureKeyVaultHandler(vaultURL, file string, config AzureConfiguration) (map[string]string, error) {
	var secretVars []string
	secrets := make(map[string]string)

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
		split := strings.Split(v, "=")
		key := split[0]
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
			secrets[keyAsProp] = secret
		}
	}

	return secrets, nil
}

// NewKeyVaultClient creates a new keyvault client
func newKeyVaultClient(vaultURL string, config AzureConfiguration) (*KeyVault, error) {

	keyClient := keyvault.New()
	credentials := azureAuth.NewClientCredentialsConfig(config.ClientID, config.ClientSecret, config.TenantID)
	// set the correct resource for keyvault
	credentials.Resource = "https://vault.azure.net"
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
