package yaml

import (
	"fmt"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"

	"gopkg.in/yaml.v3"
)

type YAMLFormatter struct {
}

type YAMLConverter struct {
}

func (p *YAMLFormatter) Format(data interface{}) (string, error) {
	formatted, err := yaml.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("error during parsing of data for output type: %s",
			cfg.Config.Output)
	}

	return string(formatted), nil
}

func (p *YAMLConverter) ToMap(data []byte) (interface{}, error) {
	var result map[string]interface{}
	err := yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, fmt.Errorf("error occurred during parse of object to map: %s : %s",
			string(data), err.Error())
	}

	return result, nil
}
