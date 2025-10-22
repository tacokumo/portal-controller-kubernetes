package helmutil

import "go.yaml.in/yaml/v2"

func StructToValueMap(i any) (map[string]interface{}, error) {
	data, err := yaml.Marshal(i)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	err = yaml.Unmarshal(data, &result)
	if err != nil {
		return nil, err
	}

	return result, nil
}
