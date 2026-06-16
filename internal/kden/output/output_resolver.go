package output

import (
	"fmt"

	"strings"

	cfg "github.com/konfidence-project/konfidence/internal/kden/config"

	"github.com/konfidence-project/konfidence/internal/kden/log"
	"github.com/konfidence-project/konfidence/internal/kden/output/json"
	"github.com/konfidence-project/konfidence/internal/kden/output/pretty"
	"github.com/konfidence-project/konfidence/internal/kden/output/yaml"
)

type FormattedOutput string

const (
	JsonOutputFormat        FormattedOutput = "json"
	YamlOutputFormat        FormattedOutput = "yaml"
	TablePrettyOutputFormat FormattedOutput = "pretty"
)

type Formatter interface {
	Format(data interface{}) (string, error)
}

type Converter interface {
	ToMap(data []byte) (interface{}, error)
}

var Formatters = map[FormattedOutput]Formatter{
	JsonOutputFormat: &json.JSONFormatter{},
	YamlOutputFormat: &yaml.YAMLFormatter{},
}

var Converters = map[FormattedOutput]Converter{
	JsonOutputFormat: &json.JSONConverter{},
	YamlOutputFormat: &yaml.YAMLConverter{},
}

func ResolveFormat(data interface{}, command string) (string, error) {
	outputFormat := FormattedOutput(strings.ToLower(cfg.Config.Output))

	if data == nil {
		return "", fmt.Errorf("error while resolving output format with no data")
	}

	if outputFormat == TablePrettyOutputFormat {
		log.Info("Using table pretty output format")
		modelFunc, ok := pretty.GetModelFuncMap()[command]
		if !ok {
			return "", fmt.Errorf("error while resolving command %s invocation", command)
		}

		return pretty.FormatTable(modelFunc, data)
	}

	formatted, ok := Formatters[outputFormat]
	if !ok {
		return "", fmt.Errorf("error with provided output format: %s", outputFormat)
	}

	if encodedObject, isBytes := data.([]byte); isBytes {
		converter := Converters[outputFormat]
		object, err := converter.ToMap(encodedObject)
		if err != nil {
			return "", err
		}
		return formatted.Format(object)
	}

	return formatted.Format(data)
}

func PrintMessage(msg string) {
	fmt.Print(msg)
}
