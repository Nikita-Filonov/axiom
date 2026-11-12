package teststats

import (
	"encoding/json"
	"errors"
	"fmt"
)

// Attempt documents contain neither null values nor duplicate object keys.
func validateJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token == nil {
		return errors.New("attempt fields must not contain JSON null")
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return nil
	}
	keys := make(map[string]bool)
	for decoder.More() {
		if delimiter == '{' {
			key, err := decoder.Token()
			if err != nil {
				return err
			}
			// Token returns string keys inside an object; malformed keys return an error.
			name := key.(string)
			if keys[name] {
				return fmt.Errorf("duplicate JSON key %q", name)
			}
			keys[name] = true
		}
		if err := validateJSONValue(decoder); err != nil {
			return err
		}
	}
	_, err = decoder.Token()
	return err
}
