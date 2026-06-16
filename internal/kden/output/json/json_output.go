package json

import (
	"encoding/json"
	"fmt"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"

	"gopkg.in/yaml.v3"
)

type JSONFormatter struct {
}

type JSONConverter struct {
}

func (p *JSONFormatter) Format(data interface{}) (string, error) {
	formatted, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return "", fmt.Errorf("error during parsing of data for output type: %s",
			cfg.Config.Output)
	}

	return string(formatted), nil
}

func (p *JSONConverter) ToMap(data []byte) (interface{}, error) {
	var result map[string]interface{}
	// YAML unmarshalling can handle the correct output type regardless of input type.
	// In order to avoid explicit definition of the input type, we use it instead of JSON unmarshalling
	err := yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("error occurred during parse of object to map: %s : %s",
			string(data), err.Error())
	}

	return result, nil
}
