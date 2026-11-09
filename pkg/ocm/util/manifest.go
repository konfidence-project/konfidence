package util

import (
	"bytes"
	"encoding/json"
	"errors"

	"sigs.k8s.io/yaml"
)

// UnmarshalManifest reads a Konfidence manifest blob, which is either JSON or YAML.
func UnmarshalManifest(data []byte, v any) error {
	if json.Valid(data) {
		return json.Unmarshal(data, v)
	}
	converted, err := yaml.YAMLToJSON(data)
	if err != nil {
		return err
	}
	if bytes.Equal(converted, []byte("null")) {
		return errors.New("manifest is empty")
	}
	return json.Unmarshal(converted, v)
}
