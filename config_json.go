package lenses

import (
	"bytes"
	"encoding/json"
	"fmt"
)

var commaSep = []byte(",")

// ConfigMarshalJSON returns the JSON encoding of "c" `Config`.
func ConfigMarshalJSON(c Config) ([]byte, error) {
	if len(c.Contexts) == 0 {
		return nil, fmt.Errorf("json write: contexts can not be empty")
	}

	if c.CurrentContext == "" {
		c.CurrentContext = DefaultContextKey
	}

	result := new(bytes.Buffer)

	// write current context.
	result.WriteString(fmt.Sprintf(`{"%s":"%s",`, currentContextKeyJSON, c.CurrentContext))

	// write contexts.
	result.WriteString(fmt.Sprintf(`"%s":{`, contextsKeyJSON))

	// loop over contexts found and append its result.
	n := 0
	for contextKey, v := range c.Contexts {
		n++
		b, err := ClientConfigMarshalJSON(*v)
		if err != nil {
			return nil, fmt.Errorf("json write: error writing the context '%s': %v", contextKey, err)
		}

		result.WriteString(fmt.Sprintf(`"%s":`, contextKey))
		result.Write(b)

		if n < len(c.Contexts) {
			result.Write(commaSep)
		}
	}

	result.WriteString("}") // end of contexts map.
	result.WriteString("}") // end of json obj.

	return result.Bytes(), nil
}

// ConfigUnmarshalJSON parses the JSON-encoded `Config` and stores the result
// in the `Config` pointed to by "c".
func ConfigUnmarshalJSON(b []byte, c *Config) error {
	var keys map[string]json.RawMessage
	err := json.Unmarshal(b, &keys)
	if err != nil {
		return err
	}

	// check if contains a valid authentication key.
	for key, value := range keys {
		if key == currentContextKeyJSON {
			if err := json.Unmarshal(value, &c.CurrentContext); err != nil {
				return fmt.Errorf("json: current context unmarshal: %v", err)
			}
			continue
		}

		if key == contextsKeyJSON {
			if c.Contexts == nil {
				c.Contexts = make(map[string]*ClientConfig)
			}

			bb, err := value.MarshalJSON()
			if err != nil {
				return fmt.Errorf("json: context unmarshal: %v", err)
			}

			if len(bb) == 0 {
				return fmt.Errorf("json: contexts can not be empty")
			}

			var contextsJSON map[string]json.RawMessage

			if err := json.Unmarshal(bb, &contextsJSON); err != nil {
				return err
			}

			for k, v := range contextsJSON {
				var clientConfig ClientConfig
				if err := ClientConfigUnmarshalJSON(v, &clientConfig); err != nil {
					return err // exit on first failure.
				}

				c.Contexts[k] = &clientConfig
			}
		}
	}

	return nil
}

var bracketRightB = []byte("}")

// ClientConfigMarshalJSON retruns the json string as bytes of the given `ClientConfig` structure.
func ClientConfigMarshalJSON(c ClientConfig) ([]byte, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	if c.Authentication == nil {
		return nil, nil
	}

	var (
		authenticationKey string
		content           []byte
	)

	switch auth := c.Authentication.(type) {
	case BasicAuthentication:
		content, err = json.Marshal(auth)
		if err != nil {
			return nil, err
		}
		authenticationKey = basicAuthenticationKeyJSON
	case KerberosAuthentication:
		content, err = kerberosAuthenticationMarshalJSON(auth)
		if err != nil {
			return nil, err
		}
		authenticationKey = kerberosAuthenticationKeyJSON
	}

	content = append(append(commaSep, []byte(fmt.Sprintf(`"%s":`, authenticationKey))...), content...)
	// b = bytes.Replace(b, bracketRightB, append(content, commaSep...), 1)
	b = bytes.Replace(b, bracketRightB, append(content, bracketRightB...), 1)
	return b, nil
}

var rightBrace byte = '}'

func kerberosAuthenticationMarshalJSON(auth KerberosAuthentication) ([]byte, error) {
	if auth.Method == nil {
		return nil, fmt.Errorf("json write: kerberos authentication: method missing")
	}

	// {"confFile":"/etc/krb5.conf"}
	b, err := json.Marshal(auth)
	if err != nil {
		return nil, err
	}

	// {"username":"testuser","password":"testpassword","realm":"my.default"}
	content, err := json.Marshal(auth.Method)
	if err != nil {
		return nil, err
	}

	var methodKey string

	switch auth.Method.(type) {
	case KerberosWithPassword:
		methodKey = kerberosWithPasswordMethodKeyJSON
	case KerberosWithKeytab:
		methodKey = kerberosWithKeytabMethodKeyJSON
	case KerberosFromCCache:
		methodKey = kerberosFromCCacheMethodKeyJSON
	}

	// ,"withPassword":{"username":"testuser","password":"testpassword","realm":"my.default"}
	content = append(append(commaSep, []byte(fmt.Sprintf(`"%s":`, methodKey))...), content...)

	// {"confFile":"/etc/krb5.conf","withPassword":{"username":"testuser","password":"testpassword","realm":"my.default"}}
	b = append(b[0:len(b)-1], append(content, rightBrace)...)
	return b, nil
}

// ClientConfigUnmarshalJSON parses the JSON-encoded `ClientConfig` and stores the result
// in the `ClientConfig` pointed to by "c".
func ClientConfigUnmarshalJSON(b []byte, c *ClientConfig) error {
	// first unmarshal the known types.
	if err := json.Unmarshal(b, c); err != nil {
		return err
	}

	// second, get all.
	var raw map[string]json.RawMessage
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	for k, v := range raw {
		isBasicAuth := k == basicAuthenticationKeyJSON
		isKerberosAuth := k == kerberosAuthenticationKeyJSON
		if isBasicAuth || isKerberosAuth {
			bb, err := v.MarshalJSON()
			if err != nil {
				return err
			}

			if isBasicAuth {
				var auth BasicAuthentication
				if err = json.Unmarshal(bb, &auth); err != nil {
					return err
				}
				c.Authentication = auth
				return nil
			}

			var auth KerberosAuthentication
			if err = kerberosAuthenticationUnmarshalJSON(bb, &auth); err != nil {
				return err
			}
			c.Authentication = auth
			return nil
		}
	}

	// no new format found, let's do a backwards compatibility for "user" and "password" fields -> BasicAuthentication.
	if usernameJSON, passwordJSON := raw["user"], raw["password"]; len(usernameJSON) > 0 && len(passwordJSON) > 0 {
		// need to escape those "\"...\"".
		var auth BasicAuthentication
		if err := json.Unmarshal(usernameJSON, &auth.Username); err != nil {
			return err
		}

		if err := json.Unmarshal(passwordJSON, &auth.Password); err != nil {
			return err
		}

		c.Authentication = auth
		return nil
	}

	return fmt.Errorf("json: unknown or missing authentication key")
}

func kerberosAuthenticationUnmarshalJSON(b []byte, auth *KerberosAuthentication) error {
	var raw map[string]json.RawMessage
	err := json.Unmarshal(b, &raw)
	if err != nil {
		return err
	}

	for key, value := range raw {
		if key == kerberosConfFileKeyJSON {
			if err = json.Unmarshal(value, &auth.ConfFile); err != nil {
				return err
			}
			continue
		}

		bb, err := value.MarshalJSON()
		if err != nil {
			return err
		}

		switch key {
		case kerberosWithPasswordMethodKeyJSON:
			var method KerberosWithPassword
			if err = json.Unmarshal(bb, &method); err != nil {
				return err
			}
			auth.Method = method
		case kerberosWithKeytabMethodKeyJSON:
			var method KerberosWithKeytab
			if err = json.Unmarshal(bb, &method); err != nil {
				return err
			}
			auth.Method = method
		case kerberosFromCCacheMethodKeyJSON:
			var method KerberosFromCCache
			if err = json.Unmarshal(bb, &method); err != nil {
				return err
			}
			auth.Method = method
		default:
			return fmt.Errorf("json: unexpected key: %s", key)
		}
	}

	if auth.Method == nil {
		return fmt.Errorf("json: kerberos: no authentication method found inside")
	}

	return nil
}
