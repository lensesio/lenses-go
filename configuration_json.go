package lenses

import (
	"bytes"
	"encoding/json"
	"fmt"
)

var commaSep = []byte(",")

// ConfigurationMarshalJSON returns the JSON encoding of "c" `Configuration`.
func ConfigurationMarshalJSON(c Configuration) ([]byte, error) {
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
		b, err := clientConfigurationMarshalJSON(*v)
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

// ConfigurationUnmarshalJSON parses the JSON-encoded `Configuration` and stores the result
// in the `Configuration` pointed to by "c".
func ConfigurationUnmarshalJSON(b []byte, c *Configuration) error {
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
				c.Contexts = make(map[string]*ClientConfiguration)
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
				var clientConfig ClientConfiguration
				if err := clientConfigurationUnmarshalJSON(v, &clientConfig); err != nil {
					return err // exit on first failure.
				}

				c.Contexts[k] = &clientConfig
			}
		}
	}

	return nil
}

// clientConfigurationMarshalJSON retruns the json string as bytes of the given `ClientConfiguration` structure.
func clientConfigurationMarshalJSON(c ClientConfiguration) ([]byte, error) {
	b, err := json.Marshal(c)
	if err != nil {
		return nil, err
	}

	if c.Authentication == nil {
		return nil, nil
	}

	switch auth := c.Authentication.(type) {
	case BasicAuthentication:
		bb, err := json.Marshal(auth)
		if err != nil {
			return nil, err
		}
		bb = append(append(commaSep, []byte(fmt.Sprintf(`"%s":`, basicAuthenticationKeyJSON))...), bb...)
		b = bytes.Replace(b, commaSep, append(bb, commaSep...), 1)
	case KerberosAuthentication:
	}

	return b, nil
}

func clientConfigurationUnmarshalJSON(b []byte, c *ClientConfiguration) error {
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
			if err = json.Unmarshal(bb, &auth); err != nil {
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

func kerberosAuthenticationMarshalJSON(b []byte, auth *KerberosAuthentication) error {
	return fmt.Errorf("not implemented yet")
}
