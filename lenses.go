package lenses

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/kataras/golog"
	"gopkg.in/jcmturner/gokrb5.v5/client"
	kerberosconfig "gopkg.in/jcmturner/gokrb5.v5/config"
	"gopkg.in/jcmturner/gokrb5.v5/credentials"
	"gopkg.in/jcmturner/gokrb5.v5/keytab"
)

// Version is the current semantic version of the lenses client and cli.
const Version = "2.1.0"

// ConnectionOption describes an optional runtime configurator that can be passed on `OpenConnection`.
// Custom `ConnectionOption` can be used as well, it's just a type of `func(*lenses.Client)`.
//
// Look `UsingClient` and `UsingToken` for use-cases.
type ConnectionOption func(*Client)

func getTimeout(httpClient *http.Client, timeoutStr string) time.Duration {
	// config's timeout has priority if the httpClient passed has smaller or not-seted timeout.
	timeout, _ := time.ParseDuration(timeoutStr)
	if timeout > httpClient.Timeout { // skip error, we don't care here.
		return timeout
	}

	return httpClient.Timeout
}

func getTransportLayer(httpClient *http.Client, timeout time.Duration) (t http.RoundTripper) {
	if t := httpClient.Transport; t != nil {
		return t
	}

	httpTransport := &http.Transport{
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
		// TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	if timeout > 0 {
		httpTransport.Dial = func(network string, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, timeout)
		}
	}

	return httpTransport
}

// UsingClient modifies the underline HTTP Client that lenses is using for contact with the backend server.
func UsingClient(httpClient *http.Client) ConnectionOption {
	return func(c *Client) {
		if httpClient == nil {
			return
		}

		// config's timeout has priority if the httpClient passed has smaller or not-seted timeout.
		timeout := getTimeout(httpClient, c.config.Timeout)

		transport := getTransportLayer(httpClient, timeout)
		httpClient.Transport = transport

		c.client = httpClient
	}
}

// UsingToken can specify a custom token that can by-pass the "user" and "password".
// It may be useful for testing purposes.
func UsingToken(tok string) ConnectionOption {
	return func(c *Client) {
		if tok == "" {
			return
		}

		c.config.Token = tok
	}

}

// OpenConnection creates & returns a new Landoop's Lenses API bridge interface
// based on the passed Configuration and the (optional) options.
// OpenConnection authenticates the user and returns a valid ready-to-use `*lenses.Client`.
// If failed to communicate with the server then it returns a nil client and a non-nil error.
//
// Usage:
// config := lenses.Configuration{Host: "", User: "", Password: "", Timeout: "15s"}
// client, err := lenses.OpenConnection(config) // or config, lenses.UsingClient/UsingToken
// if err != nil { panic(err) }
// client.DeleteTopic("topicName")
//
// Read more by navigating to the `Client` type documentation.
func OpenConnection(config Configuration, options ...ConnectionOption) (*Client, error) {
	c := &Client{config: config}
	for _, opt := range options {
		opt(c)
	}

	if !config.IsValid() {
		return nil, fmt.Errorf("invalid configuration: Token or (User or Password) missing")
	}

	// if client is not set-ed by any option, set it to a new one,
	// a good idea could be to use the `http.DefaultClient`
	// but this has some limitations so we start with a new, to be clear and simple.
	if c.client == nil {
		httpClient := &http.Client{}
		UsingClient(httpClient)(c)
	}

	if c.config.Token != "" {
		golog.Debugf("Connecting using just the token: %s", config.Token)
		// User will be empty but it does its job.
		return c, nil
	}

	var (
		resp *http.Response
		err  error
	)

	if c.config.User != "" && c.config.Password != "" && c.config.Kerberos.ConfFile == "" {
		// auth by raw username/password.
		// retrieve token.
		userAuthJSON := fmt.Sprintf(`{"user":"%s", "password": "%s"}`, c.config.User, c.config.Password)

		resp, err = c.do(http.MethodPost, "api/login", contentTypeJSON, []byte(userAuthJSON))
		if err != nil {
			return nil, fmt.Errorf("%s or kerberos authentication is required", err.Error())
		}

		if resp.StatusCode == http.StatusUnauthorized {
			return nil, fmt.Errorf("client: auth failure: basic: StatusUnauthorized 401")
		}

		tokenBytes, err := c.readResponseBody(resp)
		resp.Body.Close()

		if err != nil {
			return nil, err
		}

		if len(tokenBytes) == 0 {
			return nil, fmt.Errorf("client: auth failure: retrieved an empty token, please report it as bug")
		}

		resp, err = c.do(http.MethodGet, "/api/auth", "", nil, func(req *http.Request) error {
			req.Header.Set(xKafkaLensesTokenHeaderKey, string(tokenBytes))
			return nil
		})
	} else if krb5 := c.config.Kerberos; krb5.IsValid() {
		absPath, err := filepath.Abs(krb5.ConfFile)
		if err != nil {
			return nil, fmt.Errorf("client: auth failure: kerberos failure: unable to retrieve absolute file location for '%s': %v", krb5.ConfFile, err)
		}
		f, err := os.Open(absPath)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("client: auth failure: kerberos failure: unable to find conf file '%s': %v", absPath, err)
		}

		var kerberosClient client.Client
		if krb5.KeyTabFile == "" && krb5.CCacheFile == "" {
			// load via parent `User` and `Password` and its `Realm` (if it was empty, then the underline library will use the default one).
			if c.config.User == "" && c.config.Password == "" {
				return nil, fmt.Errorf("client: auth failure: kerberos failure: with password: 'Configuration#User' and 'Configuration#Password' are both required")
			}

			kerberosClient = client.NewClientWithPassword(c.config.User, krb5.Realm, c.config.Password)
		} else if krb5.KeyTabFile != "" {
			// load via keytab.
			if c.config.User == "" {
				return nil, fmt.Errorf("client: auth failure: kerberos failure: with keytab: 'Configuration#User' is required")
			}
			kt, err := keytab.Load(krb5.KeyTabFile)
			if err != nil {
				return nil, fmt.Errorf("client: auth failure: kerberos failure: with keytab: unable to load keytab file '%s': %v", krb5.KeyTabFile, err)
			}
			kerberosClient = client.NewClientWithKeytab(c.config.User, krb5.Realm, kt)
		} else {
			// load via ccache, one of these cases should be executed.
			cc, err := credentials.LoadCCache(krb5.CCacheFile)
			if err != nil {
				return nil, fmt.Errorf("client: auth failure: kerberos failure: with ccache: unable to load ccache file '%s': %v", krb5.CCacheFile, err)
			}

			kerberosClient, err = client.NewClientFromCCache(cc)
			if err != nil { // stop as soon as possible.
				return nil, fmt.Errorf("client: auth failure: kerberos failure: with ccache: %v", err)
			}
		}

		kerberosConfig, err := kerberosconfig.Load(krb5.ConfFile)
		if err != nil {
			return nil, fmt.Errorf("client: auth failure: kerberos invalid configuration: %v", err)
		}

		if err = kerberosClient.WithConfig(kerberosConfig).Login(); err != nil {
			return nil, fmt.Errorf("client: auth failure: kerberos login: %v", err)
		}

		c.persistentRequestOption = func(r *http.Request) error {
			return kerberosClient.SetSPNEGOHeader(r, fmt.Sprintf("%s/%s", "HTTP", r.URL.Hostname()))
		}

		resp, err = c.do(http.MethodGet, "/api/auth", contentTypeJSON, nil)
		if err != nil {
			return nil, fmt.Errorf("client: auth failure: kerberos failed to sent SPNEGO header: %v", err)
		}
	} else { // no, don't do it automatically, user should know if <- isKerberosConfReal("/etc/krb5.conf") {
		return nil, fmt.Errorf("client: auth failure: 'User' and 'Password' or 'Kerberos' missing from the Configuration")
	}

	if err != nil {
		return nil, err
	}

	// set the token we received.
	var loginData User
	if err := c.readJSON(resp, &loginData); err != nil {
		return nil, err
	}

	if loginData.Token == "" { // this should never happen.
		return nil, fmt.Errorf("client: login failure: token is undefinied")
	}

	if config.Debug {
		golog.SetLevel("debug")
		golog.Debugf("Connected on %s with token: %s.\nUser details: %#+v",
			c.config.Host, loginData.Token, loginData.Name)
	}

	// set the generated token and the user model retrieved from server.
	c.config.Token = loginData.Token
	c.user = loginData

	return c, nil
}
