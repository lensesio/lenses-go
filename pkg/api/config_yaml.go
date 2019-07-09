package api

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v2"
)

var newLineB = []byte("\n")

// ConfigMarshalYAML returns the YAML encoding of "c" `Config`.
func ConfigMarshalYAML(c Config) ([]byte, error) {
	if len(c.Contexts) == 0 {
		return nil, fmt.Errorf("yaml write: contexts can not be empty")
	}

	if c.CurrentContext == "" {
		c.CurrentContext = DefaultContextKey
	}

	result := new(bytes.Buffer)

	// write current context.
	result.WriteString(fmt.Sprintf("%s: %s\n", currentContextKeyYAML, c.CurrentContext))

	// write "Contexts".
	result.WriteString(fmt.Sprintf("%s:\n", contextsKeyYAML))
	n := 0
	for contextKey, clientConfig := range c.Contexts {
		n++
		b, err := ClientConfigMarshalYAML(*clientConfig)
		if err != nil {
			return nil, fmt.Errorf("yaml write: error writing the context [%s]: [%v]", contextKey, err)
		}

		result.WriteString(fmt.Sprintf("  %s:\n", contextKey))
		newLineWithMoreSpaces := append(newLineB, []byte("    ")...)
		b = bytes.Replace(b, newLineB, newLineWithMoreSpaces, -1)
		// remove trailing \n   but make a check because if it's only the context key
		// but not content below it (it can happen if user mess up with his config) it would panic.
		if len(b) > len(newLineWithMoreSpaces)+1 {
			b = b[0 : len(b)-len(newLineWithMoreSpaces)]
		}

		result.Write(append([]byte("    "), b...))

		if n < len(c.Contexts) {
			result.Write(newLineB)
		}
	}
	return result.Bytes(), nil
}

// ClientConfigMarshalYAML retruns the yaml string as bytes of the given `ClientConfig` structure.
func ClientConfigMarshalYAML(c ClientConfig) ([]byte, error) {
	if c.Authentication == nil {
		return nil, nil
	}

	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}

	var (
		authenticationKey string
		content           []byte
	)

	switch auth := c.Authentication.(type) {
	case BasicAuthentication:
		content, err = yaml.Marshal(auth) // basic auth is ok, doesn't contain any nested interface.
		if err != nil {
			return nil, err
		}
		authenticationKey = basicAuthenticationKeyYAML
	case KerberosAuthentication:
		content, err = kerberosAuthenticationMarshalYAML(auth)
		if err != nil {
			return nil, err
		}
		authenticationKey = kerberosAuthenticationKeyYAML
	}

	content = toYAMLNode(content)
	b = append(b, append(append([]byte(fmt.Sprintf(`%s:`, authenticationKey)), newLineWithSpaces...), content...)...)

	return b, nil
}

var newLineWithSpaces = append(newLineB, []byte("  ")...)

func toYAMLNode(content []byte) []byte {
	newLinesCount := bytes.Count(content, newLineB) // no trailing new line.
	return bytes.Replace(content, newLineB, newLineWithSpaces, newLinesCount-1)
}

func kerberosAuthenticationMarshalYAML(auth KerberosAuthentication) ([]byte, error) {
	if auth.Method == nil {
		return nil, fmt.Errorf("yaml write: kerberos authentication: method missing")
	}

	b, err := yaml.Marshal(auth)
	if err != nil {
		return nil, err
	}

	content, err := yaml.Marshal(auth.Method)
	if err != nil {
		return nil, err
	}

	var methodKey string

	switch auth.Method.(type) {
	case KerberosWithPassword:
		methodKey = kerberosWithPasswordMethodKeyYAML
	case KerberosWithKeytab:
		methodKey = kerberosWithKeytabMethodKeyYAML
	case KerberosFromCCache:
		methodKey = kerberosFromCCacheMethodKeyYAML
	}

	content = toYAMLNode(content)
	b = append(b, append(append([]byte(fmt.Sprintf(`%s:`, methodKey)), newLineWithSpaces...), content...)...)

	return b, nil
}

// ConfigUnmarshalYAML parses the YAML-encoded `Config` and stores the result
// in the `Config` pointed to by "c".
func ConfigUnmarshalYAML(b []byte, c *Config) error {
	var tree yaml.MapSlice
	err := yaml.Unmarshal(b, &tree)
	if err != nil {
		return err
	}

	for _, item := range tree {
		key, ok := item.Key.(string)
		if !ok {
			return fmt.Errorf("yaml: expected '%v' key to be string", item.Key)
		}

		if item.Value == nil {
			continue
		}

		if key == currentContextKeyYAML {
			contextKey := DefaultContextKey
			if contextKeyValue, ok := item.Value.(string); ok {
				contextKey = contextKeyValue
			}

			c.CurrentContext = contextKey
			continue
		}

		if key == contextsKeyYAML {
			contextsKeyTree, ok := item.Value.(yaml.MapSlice)
			if !ok || len(contextsKeyTree) == 0 {
				return fmt.Errorf("yaml: unable to unmarshal contexts, not a valid map type")
			}

			if c.Contexts == nil {
				c.Contexts = make(map[string]*ClientConfig)
			}

			/*
				yaml.MapSlice{
					yaml.MapItem{Key:"master", Value:yaml.MapSlice{
						yaml.MapItem{Key:"Host", Value:"https://landoop.com"},
						yaml.MapItem{Key:"Basic", Value:yaml.MapSlice{
							yaml.MapItem{Key:"Username", Value:"testuser"},
							yaml.MapItem{Key:"Password", Value:"testpassword"}}},
						yaml.MapItem{Key:"Timeout", Value:"11s"},
						yaml.MapItem{Key:"Debug", Value:true}}
				}}
			*/
			for _, contextKeyItem := range contextsKeyTree {
				// yaml.MapItem{Key:"master",
				contextKey, ok := contextKeyItem.Key.(string)
				if !ok {
					return fmt.Errorf("yaml: context key should be a string")
				}

				//  Value:yaml.MapSlice{
				contextTree, ok := contextKeyItem.Value.(yaml.MapSlice)
				if !ok || len(contextTree) == 0 {
					return fmt.Errorf("yaml: unable to unmarshal context [%s], not a valid map type", contextKey)
				}

				clientConfig := new(ClientConfig)
				bb, err := yaml.Marshal(contextTree)
				if err != nil {
					return err
				}

				if err = yaml.Unmarshal(bb, clientConfig); err != nil {
					return err
				}

				// now for authentication keys, manually.
				for _, contextPropertyItem := range contextTree {
					// yaml.MapItem{Key:"Host", Value:"..."}
					propertyKey, ok := contextPropertyItem.Key.(string)
					if !ok {
						return fmt.Errorf("yaml: expected property key [%v] to be a string", contextPropertyItem.Key)
					}

					isBasicAuth := propertyKey == basicAuthenticationKeyYAML
					isKerberosAuth := propertyKey == kerberosAuthenticationKeyYAML
					if isBasicAuth || isKerberosAuth { // should be one of those.
						bb, err = yaml.Marshal(contextPropertyItem.Value)
						if err != nil {
							return err
						}

						if isBasicAuth {
							var auth BasicAuthentication
							if err = yaml.Unmarshal(bb, &auth); err != nil {
								return err
							}
							clientConfig.Authentication = auth
							continue
						}

						var auth KerberosAuthentication
						if err = kerberosAuthenticationUnmarshalYAML(bb, &auth); err != nil {
							return err
						}
						clientConfig.Authentication = auth
						continue
					}
				}

				// no new format found, let's do a loop again to do a backwards compatibility check for "User" and "Password" fields -> BasicAuthentication.
				var username, password string

				for _, contextPropertyItem := range contextTree {
					if username != "" && password != "" {
						break
					}

					switch contextPropertyItem.Key.(string) {
					case "User":
						username, _ = contextPropertyItem.Value.(string) // safe set.
					case "Password":
						password, _ = contextPropertyItem.Value.(string) // safe set.
					}
				}

				// both must set in order to be a valid BasicAuthentication.
				if username != "" && password != "" {
					clientConfig.Authentication = BasicAuthentication{Username: username, Password: password}
				}

				if clientConfig.Authentication == nil {
					// don't allow empty auth ofc.
					return fmt.Errorf("yaml: unknown or missing authentication key for context [%s]", contextKey)
				}

				c.Contexts[contextKey] = clientConfig
			}

		}
	}

	return nil
}

func kerberosAuthenticationUnmarshalYAML(b []byte, auth *KerberosAuthentication) error {
	var tree yaml.MapSlice
	err := yaml.Unmarshal(b, &tree)
	if err != nil {
		return err
	}

	for _, item := range tree {
		key, ok := item.Key.(string)
		if !ok {
			return fmt.Errorf("yaml: expected [%v] key to be string", item.Key)
		}

		if key == kerberosConfFileKeyYAML {
			auth.ConfFile, _ = item.Value.(string)
			continue
		}

		methodField, ok := item.Value.(yaml.MapSlice)
		if !ok {
			// we search for a `MapSlice` which will contain the method's properties first,
			// next will check if the key is a valid one too.
			continue
		}

		bb, err := yaml.Marshal(methodField)
		if err != nil {
			return err
		}

		switch key {
		case kerberosWithPasswordMethodKeyYAML:
			var method KerberosWithPassword
			if err = yaml.Unmarshal(bb, &method); err != nil {
				return err
			}
			auth.Method = method
		case kerberosWithKeytabMethodKeyYAML:
			var method KerberosWithKeytab
			if err = yaml.Unmarshal(bb, &method); err != nil {
				return err
			}
			auth.Method = method
		case kerberosFromCCacheMethodKeyYAML:
			var method KerberosFromCCache
			if err = yaml.Unmarshal(bb, &method); err != nil {
				return err
			}
			auth.Method = method
		default:
			return fmt.Errorf("yaml: unexpected key: [%s]", key)
		}
	}

	if auth.Method == nil {
		return fmt.Errorf("yaml: kerberos: no authentication method found inside")
	}

	return nil
}
