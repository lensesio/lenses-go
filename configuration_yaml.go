package lenses

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v2"
)

var newLineB = []byte("\n")

// ConfigurationMarshalYAML returns the YAML encoding of "c" `Configuration`.
func ConfigurationMarshalYAML(c Configuration) ([]byte, error) {
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
		b, err := clientConfigurationMarshalYAML(*clientConfig)
		if err != nil {
			return nil, fmt.Errorf("yaml write: error writing the context '%s': %v", contextKey, err)
		}

		result.WriteString(fmt.Sprintf("  %s:\n", contextKey))
		b = bytes.Replace(b, newLineB, append(newLineB, []byte("    ")...), -1)
		result.Write(append([]byte("    "), b...))

		if n < len(c.Contexts) {
			result.Write(newLineB)
		}
	}
	return result.Bytes(), nil
}

// clientConfigurationMarshalYAML retruns the yaml string as bytes of the given `ClientConfiguration` structure.
func clientConfigurationMarshalYAML(c ClientConfiguration) ([]byte, error) {
	b, err := yaml.Marshal(c)
	if err != nil {
		return nil, err
	}

	if c.Authentication == nil {
		return nil, nil
	}

	switch auth := c.Authentication.(type) {
	case BasicAuthentication:
		bb, err := yaml.Marshal(auth)
		if err != nil {
			return nil, err
		}

		newLinesCount := bytes.Count(bb, newLineB) // no trailing new line.
		bb = bytes.Replace(bb, newLineB, append(newLineB, []byte("  ")...), newLinesCount-1)
		b = append(b, append(append([]byte(fmt.Sprintf(`%s:`, basicAuthenticationKeyYAML)), []byte("\n  ")...), bb...)...)
	case KerberosAuthentication:
	}

	return b, nil
}

// ConfigurationUnmarshalYAML parses the YAML-encoded `Configuration` and stores the result
// in the `Configuration` pointed to by "c".
func ConfigurationUnmarshalYAML(b []byte, c *Configuration) error {
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
				c.Contexts = make(map[string]*ClientConfiguration)
			}

			/*
				yaml.MapSlice{
					yaml.MapItem{Key:"master", Value:yaml.MapSlice{
						yaml.MapItem{Key:"Host", Value:"https://landoop.com"},
						yaml.MapItem{Key:"BasicAuthentication", Value:yaml.MapSlice{
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
					return fmt.Errorf("yaml: unable to unmarshal context '%s', not a valid map type", contextKey)
				}

				clientConfig := new(ClientConfiguration)
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
						return fmt.Errorf("yaml: expected property key '%v' to be a string", contextPropertyItem.Key)
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
						if err = yaml.Unmarshal(bb, &auth); err != nil {
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
						username, ok = contextPropertyItem.Value.(string) // safe set.
					case "Password":
						password, ok = contextPropertyItem.Value.(string) // safe set.
					}
				}

				// both must set in order to be a valid BasicAuthentication.
				if username != "" && password != "" {
					clientConfig.Authentication = BasicAuthentication{Username: username, Password: password}
				}

				if clientConfig.Authentication == nil {
					// don't allow empty auth ofc.
					return fmt.Errorf("yaml: unknown or missing authentication key for context '%s'", contextKey)
				}

				c.Contexts[contextKey] = clientConfig
			}

		}
	}

	return nil
}

func kerberosAuthenticationMarshalYAML(b []byte, auth *KerberosAuthentication) error {
	return fmt.Errorf("not implemented yet")
}
