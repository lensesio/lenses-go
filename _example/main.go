package main

import "github.com/lensesio/lenses-go"

func main() {
	// Prepare authentication using raw Username and Password.
	auth := lenses.BasicAuthentication{Username: "user", Password: "pass"}
	// Or authenticate using one of those three Kerberos built'n supported authentication methods.
	/*
		kerberosAuthWithPass := lenses.KerberosAuthentication{
		    ConfFile: "/etc/krb5.conf",
		    Method:   lenses.KerberosWithPassword{Realm: "my.realm or default if empty", Username: "user", Password: "pass"},
		}

		kerberosAuthWithKeytab := lenses.KerberosAuthentication{
		    ConfFile: "/etc/krb5.conf",
		    Method:   lenses.KerberosWithKeytab{KeytabFile: "/home/me/krb5_my_keytab.txt"},
		}

		kerberosFromCCacheAuth := lenses.KerberosAuthentication{
		    ConfFile: "/etc/krb5.conf",
		    Method:   lenses.KerberosFromCCache{CCacheFile: "/tmp/krb5_my_cache_file.conf"},
		}

		Custom auth can be implement as well: `Authenticate(client *lenses.Client) error`
	*/

	// Prepare the client's configuration based on the host and the authentication above.
	clientConfig := lenses.ClientConfig{Host: "domain.com", Authentication: auth, Timeout: "15s", Debug: true}

	// Creating the client using the configuration.
	client, err := lenses.OpenConnection(clientConfig) // or (config, lenses.UsingClient(customClient)/UsingToken(ready token string))
	if err != nil {
		// handle error.
		panic(err)
	}

	// Using a client's method to do API calls.
	// All lenses-go methods return a typed value based on the call
	// and an error as second output so you can catch any error coming from backend or client, forget panics.
	// Go types are first class citizens here, we will not confuse you or let you work based on luck!
	topics, err := client.GetTopics()
	// Example on how deeply we make the difference here:
	// `Client#GetTopics` returns `[]lenses.Topic`, so you can work safely.
	// topics[index].ConsumersGroup[index].Coordinator.Host
	if err != nil {
		// handle error.
	}

	// Print the length of the topics we've just received from our Lenses Box.
	print(len(topics))
}
