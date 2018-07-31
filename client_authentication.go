package lenses

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"gopkg.in/jcmturner/gokrb5.v5/client"
	"gopkg.in/jcmturner/gokrb5.v5/config"
	"gopkg.in/jcmturner/gokrb5.v5/credentials"
	"gopkg.in/jcmturner/gokrb5.v5/keytab"
)

type (
	// Authentication is an interface which all authentication methods should implement.
	//
	// See `BasicAuthentication` and `KerberosAuthentication` too.
	Authentication interface {
		// Auth accepts the current client and returns a not-nil error if authentication failed, otherwise
		// the authentication can alter the Client to do "something" before of each request.
		Auth(c *Client) error
	}
	// AuthenticationFunc implements the Authentication, it can be used for single-line custom authentication methods.
	AuthenticationFunc func(*Client) error
)

// Auth implements the `Authentication` interface, it just calls the func.
func (auth AuthenticationFunc) Auth(c *Client) error {
	return auth(c)
}

var (
	_ Authentication = BasicAuthentication{}
	_ Authentication = KerberosAuthentication{}
)

// BasicAuthentication for Lenses, accepts raw username and password.
//
// Use it when Lenses setup with "BASIC" or "LDAP" authentication.
type BasicAuthentication struct {
	Username string `json:"username" yaml:"Username" survey:"username"`
	Password string `json:"password,omitempty" yaml:"Password" survey:"password"`
}

// Auth implements the `Authentication` for the `BasicAuthentication`.
func (auth BasicAuthentication) Auth(c *Client) error {
	// auth by raw username/password.
	if auth.Username == "" || auth.Password == "" {
		return fmt.Errorf("basic failure: 'Username' and 'Password' are both required")
	}

	// retrieve token.
	userAuthJSON := fmt.Sprintf(`{"user":"%s", "password": "%s"}`, auth.Username, auth.Password)

	resp, err := c.Do(http.MethodPost, "api/login", contentTypeJSON, []byte(userAuthJSON))
	if err != nil {
		return fmt.Errorf("%s or kerberos authentication is required", err.Error())
	}

	tokenBytes, err := c.ReadResponseBody(resp)
	resp.Body.Close()

	if err != nil {
		return err
	}

	if len(tokenBytes) == 0 {
		return fmt.Errorf("basic failure: retrieved an empty token")
	}

	resp, err = c.Do(http.MethodGet, "/api/auth", "", nil, func(req *http.Request) error {
		req.Header.Set(xKafkaLensesTokenHeaderKey, string(tokenBytes))
		return nil
	})

	if err != nil {
		return fmt.Errorf("basic failure: %v", err)
	}

	if err = c.ReadJSON(resp, &c.User); err != nil {
		return err
	}

	c.Config.Token = c.User.Token // ...
	return nil
}

// KerberosAuthentication can be used as alternative option of the `BasicAuthentication` for a more secure way to connect to the lenses backend box.
type KerberosAuthentication struct {
	ConfFile string                       `json:"confFile" yaml:"ConfFile" survey:"-"` // keep those, useful for marshal.
	Method   KerberosAuthenticationMethod `json:"-" yaml:"-" survey:"-"`
}

// WithPassword reports whether the kerberos authentication is with username, password (and realm).
func (auth KerberosAuthentication) WithPassword() (KerberosWithPassword, bool) {
	method, isWithPassword := auth.Method.(KerberosWithPassword)
	return method, isWithPassword
}

// WithKeytab reports whether the kerberos authentication is with a keytab file, username (and realm).
func (auth KerberosAuthentication) WithKeytab() (KerberosWithKeytab, bool) {
	method, isWithKeytab := auth.Method.(KerberosWithKeytab)
	return method, isWithKeytab
}

// FromCCache reports whether the kerberos authentication is loaded from a ccache file.
func (auth KerberosAuthentication) FromCCache() (KerberosFromCCache, bool) {
	method, isFromCCache := auth.Method.(KerberosFromCCache)
	return method, isFromCCache
}

// Auth implements the `Authentication` for the `KerberosAuthentication`.
func (auth KerberosAuthentication) Auth(c *Client) error {
	if auth.Method == nil {
		return fmt.Errorf("kerberos failure: authentication method is nil")
	}

	absPath, err := filepath.Abs(auth.ConfFile)
	if err != nil {
		return fmt.Errorf("kerberos failure: unable to retrieve absolute file location for '%s': %v", auth.ConfFile, err)
	}
	f, err := os.Open(absPath)
	f.Close()
	if err != nil {
		return fmt.Errorf("kerberos failure: unable to find conf file '%s': %v", absPath, err)
	}

	kerberosConfig, err := config.Load(auth.ConfFile)
	if err != nil {
		return fmt.Errorf("kerberos failure: invalid configuration: %v", err)
	}

	kc, err := auth.Method.NewClient()
	if err != nil {
		return fmt.Errorf("kerberos failure: %v", err)
	}

	kerberosClient := kc.WithConfig(kerberosConfig)

	if err = kerberosClient.Login(); err != nil {
		return fmt.Errorf("kerberos failure: login: %v", err)
	}

	c.PersistentRequestModifier = func(r *http.Request) error {
		return kerberosClient.SetSPNEGOHeader(r, fmt.Sprintf("%s/%s", "HTTP", r.URL.Hostname()))
	}

	resp, err := c.Do(http.MethodGet, "/api/auth", contentTypeJSON, nil)
	if err != nil {
		return fmt.Errorf("kerberos failure: unable to send SPNEGO header: %v", err)
	}

	if err = c.ReadJSON(resp, &c.User); err != nil {
		return err
	}

	c.Config.Token = c.User.Token // update the config's one as well for any case.
	return nil
}

// KerberosAuthenticationMethod is the interface which all available kerberos authentication methods are implement.
//
// See `KerberosWithPassword`, `KerberosWithKeytab` and `KerberosFromCCache` for more.
type KerberosAuthenticationMethod interface {
	NewClient() (client.Client, error)
}

var (
	_ KerberosAuthenticationMethod = KerberosWithPassword{}
	_ KerberosAuthenticationMethod = KerberosWithKeytab{}
	_ KerberosAuthenticationMethod = KerberosFromCCache{}
)

// KerberosWithPassword is a `KerberosAuthenticationMethod` using a username, password and optionally a realm.
//
// The `KerberosAuthentication` calls its `NewClient`.
type KerberosWithPassword struct {
	Username string `json:"username" yaml:"Username" survey:"username"`
	Password string `json:"password,omitempty" yaml:"Password" survey:"password"`

	// Realm is optional, if empty then default is used.
	Realm string `json:"realm" yaml:"Realm" survey:"realm"`
}

var emptyClient client.Client

// NewClient implements the `KerberosAuthenticationMethod` for the `KerberosWithPassword`.
func (m KerberosWithPassword) NewClient() (client.Client, error) {
	if m.Username == "" || m.Password == "" {
		return emptyClient, fmt.Errorf("with password: 'Username' and 'Password' are both required")
	}

	c := client.NewClientWithPassword(m.Username, m.Realm, m.Password)
	return c, nil
}

// KerberosWithKeytab is a `KerberosAuthenticationMethod` using a username and a keytab file path and optionally a realm.
//
// The `KerberosAuthentication` calls its `NewClient`.
type KerberosWithKeytab struct {
	Username string `json:"username" yaml:"Username" survey:"username"`

	// Realm is optional, if empty then default is used.
	Realm string `json:"realm" yaml:"Realm" survey:"realm"`
	// KeytabFile the keytab file path.
	KeytabFile string `json:"keytabFile" yaml:"KeytabFile" survey:"keytab"`
}

// NewClient implements the `KerberosAuthenticationMethod` for the `KerberosWithKeytab`.
func (m KerberosWithKeytab) NewClient() (client.Client, error) {
	// load via keytab.
	if m.Username == "" {
		return emptyClient, fmt.Errorf("with keytab: 'Username' is required")
	}

	kt, err := keytab.Load(m.KeytabFile)
	if err != nil {
		return emptyClient, fmt.Errorf("with keytab: unable to load keytab file '%s': %v", m.KeytabFile, err)
	}

	c := client.NewClientWithKeytab(m.Username, m.Realm, kt)
	return c, nil
}

// KerberosFromCCache is a `KerberosAuthenticationMethod` using a ccache file path.
//
// The `KerberosAuthentication` calls its `NewClient`.
type KerberosFromCCache struct {
	// CCacheFile should be filled with the ccache file path.
	CCacheFile string `json:"ccacheFile" yaml:"CCacheFile" survey:"ccache"`
}

// NewClient implements the `KerberosAuthenticationMethod` for the `KerberosFromCCache`.
func (m KerberosFromCCache) NewClient() (client.Client, error) {
	// load from ccache.
	cc, err := credentials.LoadCCache(m.CCacheFile)
	if err != nil {
		return emptyClient, fmt.Errorf("from ccache: unable to load ccache file '%s': %v", m.CCacheFile, err)
	}

	c, err := client.NewClientFromCCache(cc)
	if err != nil {
		return emptyClient, fmt.Errorf("from ccache: %v", err)
	}

	return c, err
}
