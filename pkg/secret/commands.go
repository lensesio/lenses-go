package secret

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	azure "github.com/Azure/go-autorest/autorest/azure"
	golog "github.com/kataras/golog"
	"github.com/lensesio/lenses-go/v5/pkg/utils"
	cobra "github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var output, role, endpoint, token, fromFile, secretsFile, appSecretsFile, connectorFile, workerFile string

// NewSecretsGroupCommand creates `secrets` command
func NewSecretsGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "secrets",
		Short: `Create secret files from Hashicorp Vault or Azure Key Vault`,
		Example: `	
secrets connect vault --role lenses --token XYZ	
secrets connect azure --client-id xxxx --client-secret xxxx --tenant-id xxxxx

secrets app vault --role lenses --token XYZ	--output json --secret-file my-secrets.json
secrets app azure --client-id xxxx --client-secret xxxx --tenant-id xxxxx --output env
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.AddCommand(NewConnectGroupCommand())
	cmd.AddCommand(NewAppGroupCommand())

	return cmd
}

// NewAppGroupCommand creates `secrets app` command
func NewAppGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "app",
		Short: `Create application config file or file to source for environment variables from secrets stored in Vault or Azure KeyVault`,
		Long: `
Create application config file or file to source for environment variables from secrets stored in Vault or Azure KeyVault

Creates a JSON, YAML or text file to source environment variables from.

Secret names must only contain 0-9, a-z, A-Z, and -

For example for Azure, an environment variable SECRET_CASSANDRA_PASSWORD 
expects a secret name cassandra-password in Azure KeyVault.

export SECRET_CASSANDRA_PASSWORD=cassandra-password

For HashiCorp Vault if a variable SECRET_CASSANDRA_PASSWORD is set 
a secret is expected in Vault under the path specified by the variables
value. For example a secret in Vault:

vault kv put secret/cassandra cassandra-password=secret cassandra-user=lenses

export SECRET_CASSANDRA_PASSWORD=/secret/data/cassandra
export SECRET_CASSANDRA_USER=/secret/data/cassandra

Variables can alternatively be loaded from a file using the from-file flag.
The file contents should be in key value in the same format as the 
environment variables
		`,
		Example: `	
secrets app vault --vault-role lenses --vault-token XYZ	--vault-addr http://127.0.0.1:8200 --output env
secrets app azure --client-id xxxx --client-secret xxxx --tenant-id xxxxx --output yaml
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.PersistentFlags().StringVar(&output, "output", "env", "Output type, env for bash environment variables (source from secret-file), json or yaml")
	cmd.PersistentFlags().StringVar(&fromFile, "from-file", "", "File to variables load from instead of looking up as environment variables, separated by =")
	cmd.PersistentFlags().StringVar(&appSecretsFile, "secret-file", "secrets", "The secret file to write secrets to as key value pair")

	cmd.AddCommand(NewVaultCommand("app"))
	cmd.AddCommand(NewAzureCommand("app"))
	return cmd
}

// NewConnectGroupCommand creates `secrets connect` command
func NewConnectGroupCommand() *cobra.Command {

	cmd := &cobra.Command{
		Use:   "connect",
		Short: `Create Apache Kafka Connect config files from secrets stored in Vault, Azure KeyVault or as environment variables`,
		Long: `
Create Apache Kafka Connect config files from secrets stored in Vault 
or Azure KeyVault

Looks for environment variables prefixed with SECRET, CONNECT and 
CONNECTOR to lookup and write files. Prefix secrets vars with SECRET_, 
Apache Kafka Connect work properties with CONNECT_ and connector instance 
properties with CONNECTOR_.

Secret names must only contain 0-9, a-z, A-Z, and -

For example for Azure, an environment variable SECRET_CONNECT_CASSANDRA_PASSWORD 
expects a secret name connect-cassandra-password in Azure KeyVault.

export SECRET_CONNECTOR_CASSANDRA_PASSWORD=connect-cassandra-password

For HashiCorp Vault if a variable SECRET_CONNECT_CASSANDRA_PASSWORD is set 
a secret is expected in Vault under the path specified by the variables
value. For example a secret in Vault:

vault kv put secret/connectors/cassandra connect-cassandra-password=secret connect-cassandra-user=lenses

export SECRET_CONNECT_CASSANDRA_PASSWORD=/secret/data/connectors/cassandra
export SECRET_CONNECT_CASSANDRA_USER=/secret/data/connectors/cassandra

In secret file:
	connect.cassandra.password=secret
	connect.cassandra.user=lenses

For Environment Variables, e.g. Kubernetes secrets mounted as environment vars

	export SECRET_CONNECT_CASSANDRA_PASSWORD=secret
	export SECRET_CONNECT_CASSANDRA_USER=lenses
	
	In secret file:
		connect.cassandra.password=secret
		connect.cassandra.user=lenses

Variables can alternatively be loaded from a file using the from-file flag.
The file contents should be in key value in the same format as the 
environment variables

Connect worker secrets can be set with the prefix WORKER_CONNECT_SECRET_

.e.g.

		export WORKER_CONNECT_SECRET_SSL_TRUSTORE_LOCATION=/secret/data/connect/workers
`,
		Example: `	
secrets connect vault --vault-role lenses --vault-token XYZ	--vault-addr http://127.0.0.1:8200
secrets connect azure --vault-name lenses --client-id xxxx --client-secret xxxx --tenant-id xxxxx	
		`,
		SilenceErrors:    true,
		TraverseChildren: true,
	}

	cmd.PersistentFlags().StringVar(&secretsFile, "secret-file", "secrets.props", "The secret file to write secrets to")
	cmd.PersistentFlags().StringVar(&connectorFile, "connector-file", "connector.props", "The connector file to write connector config to")
	cmd.PersistentFlags().StringVar(&workerFile, "worker-file", "worker.props", "The connect worker file to connect worker config to")
	cmd.PersistentFlags().StringVar(&fromFile, "from-file", "", "File to variables load from instead of looking up as environment variables, separated by =")

	cmd.AddCommand(NewVaultCommand("connect"))
	cmd.AddCommand(NewAzureCommand("connect"))
	cmd.AddCommand(NewEnvCommand("connect"))

	return cmd
}

// NewVaultCommand creates `secrets connect vault` command
func NewVaultCommand(appType string) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "vault",
		Short: `Get secrets from Vault with AppRole and Kubernetes AuthMethods`,
		Long: `
Get secrets from Vault with AppRole and Kubernetes AuthMethods

Secret names must only contain 0-9, a-z, A-Z, and -

If a variable SECRET_CONNECT_CASSANDRA_PASSWORD is set 
a secret is expected in Vault under the path specified by the variables
value. For example a secret in Vault:

vault kv put secret/connectors/cassandra connect-cassandra-password=secret connect-cassandra-user=lenses

export SECRET_CONNECT_CASSANDRA_PASSWORD=/secret/data/connectors/cassandra/
export SECRET_CONNECT_CASSANDRA_USER=/secret/data/connectors/cassandra/

In secret file:
	connect.cassandra.password=secret
	connect.cassandra.user=lenses

The token is either the AppRole or Kubernetes token

Variables can alternatively be loaded from a file using the from-file flag.
The file contents should be in key value in the same format as the 
environment variables

The VAULT_ADDR, VAULT_TOKEN can also be provided as environment variables. The environment
will be checked first and an VAULT_* variables found will take precedence over the command line
options.
		`,
		Example: `
secrets connect vault --vault-role lenses --vault-token XYZ	--vault-addr http://127.0.0.1:8200
secrets app vault --vault-role lenses --vault-token XYZ	--vault-addr http://127.0.0.1:8200 --from-file my-env.txt
`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			secrets, err := VaultConnectExternalHandler(role, token, endpoint, fromFile)

			if err != nil {
				return err
			}

			if "connect" == appType {
				return writeConnectFiles(secrets, secretsFile, connectorFile, workerFile, fromFile)
			}

			if "app" == appType {
				return writeSecrets(output, secrets, appSecretsFile)
			}

			return nil
		},
	}

	cmd.PersistentFlags().StringVar(&endpoint, "vault-addr", "", "Vault server address")
	cmd.Flags().StringVar(&role, "vault-role", "", "Vault appRole name")
	cmd.Flags().StringVar(&token, "vault-token", "", "Vault or kubernetes JWT token")
	return cmd
}

// NewAzureCommand get secrets for azure by app type
func NewAzureCommand(appType string) *cobra.Command {
	var clientID, clientSecret, tenantID, dns, vaultName string

	cmd := &cobra.Command{
		Use:   "azure",
		Short: `Get secrets from Azure Key Vault`,
		Long: `
Get secrets from Azure Key Vault.

Secret names must only contain 0-9, a-z, A-Z, and -

An environment variable SECRET_CONNECT_CASSANDRA_PASSWORD 
expects a secret name connect-cassandra-password in Azure KeyVault.

export SECRET_CONNECT_CASSANDRA_PASSWORD=connect-cassandra-password

In secret file:
	connect.cassandra.password=secret
	connect.cassandra.user=lenses

Variables can alternatively be loaded from a file using the from-file flag.
The file contents should be in key value in the same format as the 
environment variables

Flags can also be set as environment variables, AZURE_CLIENT_ID, AZURE_CLIENT_SECRET,
AZURE_TENANT_ID, AZURE_KEY_VAULT and AZURE_KEY_VAULT_DNS
`,
		Example: `
secrets connect azure --vault-name lenses --client-id xxxx --client-secret xxxx --tenant-id xxxxx
secrets app azure --vault-name lenses --client-id xxxx --client-secret xxxx --tenant-id xxxxx -from-file my-env.txt
`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			azureConfig := AzureConfiguration{
				TenantID:     os.Getenv(EnvAzureClientTenantID),
				ClientID:     os.Getenv(EnvAzureClientID),
				ClientSecret: os.Getenv(EnvAzureClientSecret),
				KeyVaultName: os.Getenv(EnvAzureKeyVaultName),
			}

			if azureConfig.ClientID == "" {
				if clientID == "" {
					golog.Error(`Required flag "client-id" not set and no AZURE_CLIENT_ID environment variable found`)
					return errors.New("")
				}
				azureConfig.ClientID = clientID
			}

			if azureConfig.ClientSecret == "" {
				if clientSecret == "" {
					golog.Error(`Required flag "client-secret" not set and no AZURE_CLIENT_SECRET environment variable found`)
					os.Exit(1)
				}
				azureConfig.ClientSecret = clientSecret
			}

			if azureConfig.TenantID == "" {
				if tenantID == "" {
					golog.Error(`Required flag "tenant-id" not set and no AZURE_TENANT_ID environment variable found`)
					os.Exit(1)
				}
				azureConfig.TenantID = tenantID
			}

			if azureConfig.KeyVaultName == "" {
				if vaultName == "" {
					golog.Error(`Required flag "vault-name" not set and no AZURE_KEY_VAULT environment variable found`)
					os.Exit(1)
				}
			}

			// set the dns to the default if not provided
			if dns == "" && os.Getenv(EnvAzureKeyVaultDNS) == "" {
				dns = azure.PublicCloud.KeyVaultDNSSuffix
			}

			vaultURL := fmt.Sprintf("https://%s.%s", vaultName, dns)
			secrets, err := AzureKeyVaultHandler(vaultURL, fromFile, azureConfig)

			if err != nil {
				return err
			}

			if "connect" == appType {
				return writeConnectFiles(secrets, secretsFile, connectorFile, workerFile, fromFile)
			}

			if "app" == appType {
				return writeSecrets(output, secrets, appSecretsFile)
			}

			return nil

		},
	}

	cmd.Flags().StringVar(&vaultName, "vault-name", "", "Azure key vault name")
	cmd.Flags().StringVar(&clientID, "client-id", "", "Azure client id")
	cmd.Flags().StringVar(&clientSecret, "client-secret", "", "Azure client secret id")
	cmd.Flags().StringVar(&tenantID, "tenant-id", "", "Azure tenant id")
	cmd.Flags().StringVar(&dns, "dns-suffix", azure.PublicCloud.KeyVaultDNSSuffix, "Azure key vault dns suffix")

	return cmd
}

// NewEnvCommand secrets from environment variables
func NewEnvCommand(appType string) *cobra.Command {

	cmd := &cobra.Command{
		Use:   "env",
		Short: `Get secrets from environment`,
		Long: `
Get secrets from environment variables.

export SECRET_CONNECT_CASSANDRA_PASSWORD=secret
export SECRET_CONNECT_CASSANDRA_USER=lenses

In secret file:
	connect.cassandra.password=secret
	connect.cassandra.user=lenses

Variables can alternatively be loaded from a file using the from-file flag.
The file contents should be in key value in the same format as the 
environment variables
`,
		Example: `
secrets connect env
`,
		SilenceErrors:    true,
		TraverseChildren: true,
		RunE: func(cmd *cobra.Command, args []string) error {

			secrets, err := EnvSecretHandler(fromFile)

			if err != nil {
				return err
			}

			if "connect" == appType {
				return writeConnectFiles(secrets, secretsFile, connectorFile, workerFile, fromFile)
			}

			if "app" == appType {
				return writeSecrets(output, secrets, appSecretsFile)
			}

			return nil

		},
	}

	return cmd
}

func writeConnectFiles(secrets []Secret, secretsFile, connectorFile, workerFile, fromFile string) error {

	var secretData []string

	// lookup connector instance variables
	connectorVars, err := utils.Fetch(fromFile, "CONNECTOR_")

	if err != nil {
		return err
	}

	// lookup connect worker variables
	connectVars, err := utils.Fetch(fromFile, "CONNECT_")

	if err != nil {
		return err
	}

	// add secrets to connectorData
	for _, s := range secrets {
		record := fmt.Sprintf("%s=${file:%s:%s}", s.Key, secretsFile, s.Key)
		connectorVars = append(connectorVars, record)
	}

	// write connector files
	if len(connectorVars) > 0 {
		golog.Infof("Writing connector props to [%s]", connectorFile)
		if err := utils.WriteStringFile(connectorFile, connectorVars); err != nil {
			return err
		}
	}

	// handle connect file
	if len(connectVars) > 0 {
		golog.Infof("Writing connect worker props to [%s]", workerFile)
		connectVars = append(connectVars, "\n# External secrets")
		connectVars = append(connectVars, "config.providers=file")
		connectVars = append(connectVars, "config.providers.file.class=org.apache.kafka.common.config.provider.FileConfigProvider")

		for _, s := range secrets {
			if strings.HasPrefix(s.EnvKey, ConnectWorkerSecretPrefix) {
				connectVars = append(connectVars, fmt.Sprintf("%s=%s", s.Key, s.Value))
			}
		}

		if err := utils.WriteStringFile(workerFile, connectVars); err != nil {
			return err
		}
	}

	// format secrets
	for _, s := range secrets {
		// not connect worker secrets
		if strings.HasPrefix(s.EnvKey, SecretPreFix) {
			record := fmt.Sprintf("%s=%s", s.Key, s.Value)
			secretData = append(secretData, record)
		}
	}

	if len(secretData) > 0 {
		golog.Infof("Writing connector secrets props to [%s]", secretsFile)

		if err := utils.WriteStringFile(secretsFile, secretData); err != nil {
			return err
		}
	}

	return nil
}

func writeSecrets(output string, secrets []Secret, secretsFile string) error {
	var secretData []string
	outputType := strings.ToUpper(output)

	if outputType == "ENV" {
		golog.Infof("Writing file [%s] for sourcing as environment variables", secretsFile)
		for _, s := range secrets {
			keyAsEnv := strings.ToUpper(strings.Replace(s.Key, ".", "_", -1))
			secretData = append(secretData, fmt.Sprintf("export %s=%s", keyAsEnv, s.Value))
		}

		return utils.WriteStringFile(secretsFile, secretData)
	}

	if outputType == "YAML" || outputType == "JSON" {
		var data []byte
		var err error

		if outputType == "JSON" {
			data, err = json.Marshal(secrets)
		} else {
			data, err = yaml.Marshal(secrets)
		}

		if err != nil {
			return err
		}

		golog.Infof("Writing file [%s]", secretsFile)
		utils.WriteByteFile(secretsFile, data)
		return nil
	}

	golog.Errorf("Unsupported output [%s]. Supported types are ENV, JSON and YAML", output)
	return errors.New("Unsupported output type. Supported types are ENV, JSON and YAML")
}
