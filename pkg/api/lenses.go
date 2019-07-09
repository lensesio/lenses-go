package api

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/kataras/golog"
)

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

func getTransportLayer(httpClient *http.Client, timeout time.Duration, insecure bool) (t http.RoundTripper) {
	if t := httpClient.Transport; t != nil {
		return t
	}

	httpTransport := &http.Transport{
		// Disable HTTP/2.
		TLSNextProto: make(map[string]func(authority string, c *tls.Conn) http.RoundTripper),
	}

	if insecure {
		httpTransport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
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
		timeout := getTimeout(httpClient, c.Config.Timeout)

		transport := getTransportLayer(httpClient, timeout, c.Config.Insecure)
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

		c.Config.Token = tok
	}
}

// WithContext sets the current context, the environment to load configuration from.
//
// See the `Config` structure and the `OpenConnection` function for more.
func WithContext(contextName string) ConnectionOption {
	return func(c *Client) {
		if contextName == "" {
			contextName = DefaultContextKey
		}

		c.configFull.SetCurrent(contextName)
	}
}

// OpenConnection creates & returns a new Landoop's Lenses API bridge interface
// based on the passed `ClientConfig` and the (optional) options.
// OpenConnection authenticates the user and returns a valid ready-to-use `*lenses.Client`.
// If failed to communicate with the server then it returns a nil client and a non-nil error.
//
// Usage:
// auth := lenses.BasicAuthentication{Username: "user", Password: "pass"}
// config := lenses.ClientConfig{Host: "domain.com", Authentication: auth, Timeout: "15s"}
// client, err := lenses.OpenConnection(config) // or (config, lenses.UsingClient/UsingToken)
// if err != nil { panic(err) }
// client.DeleteTopic("topicName")
//
// Read more by navigating to the `Client` type documentation.
func OpenConnection(cfg ClientConfig, options ...ConnectionOption) (*Client, error) {
	// We accept only `ClientConfig` and not the full `Config` for use ease.
	clientConfig := &cfg

	full := &Config{
		CurrentContext: DefaultContextKey,
		Contexts: map[string]*ClientConfig{
			DefaultContextKey: clientConfig,
		},
	}

	c := &Client{configFull: full, Config: clientConfig}
	for _, opt := range options {
		opt(c)
	}

	if !clientConfig.IsValid() {
		return nil, fmt.Errorf("invalid configuration: Token or Authentication missing")
	}

	// if client is not set-ed by any option, set it to a new one,
	// a good idea could be to use the `http.DefaultClient`
	// but this has some limitations so we start with a new, to be clear and simple.
	if c.client == nil {
		httpClient := &http.Client{}
		UsingClient(httpClient)(c)
	}

	// i.e `UsingToken`.
	if clientConfig.Token != "" {
		golog.Debugf("Connecting using just the token: [%s]", clientConfig.Token)
		// User will be empty but it does its job.
		return c, nil
	}

	if clientConfig.Authentication == nil {
		return nil, fmt.Errorf("client: auth failure: authenticator missing")
	}

	if err := clientConfig.Authentication.Auth(c); err != nil {
		return nil, fmt.Errorf("client: auth failure: [%v]", err)
	}

	if c.User.Token == "" { // this should never happen.
		return nil, fmt.Errorf("client: login failure: token is undefined")
	}

	if clientConfig.Debug {
		golog.SetLevel("debug")
		golog.Debugf("Connected on [%s] with token: [%s]\nUser details: [%#+v]",
			c.Config.Host, c.User.Token, c.User)
	}

	return c, nil
}
