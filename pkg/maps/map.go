package maps

import (
	"encoding/json"
	"fmt"
)

func GetValueFromRawMap(raw []byte, key string) (interface{}, error) {
	var data map[string]interface{}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil, err
	}
	if value, ok := data[key]; ok {
		return value, nil
	}
	return nil, fmt.Errorf(`key "%s" not found`, key)
}

func CheckIfValueIsPresent(data map[string]bool, key string) bool {
	value, ok := data[key]
	return ok && value
}

func GetDistinctValues(values []string) []string {
	data := map[string]bool{}
	var distinctValues []string
	for _, v := range values {
		if _, ok := data[v]; !ok {
			data[v] = true
			distinctValues = append(distinctValues, v)
		}
	}
	return distinctValues
}
