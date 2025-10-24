package helmutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStructToValueMap_PrimitiveTypes(t *testing.T) {
	type PrimitiveStruct struct {
		IntField    int    `yaml:"intField"`
		StringField string `yaml:"stringField"`
		BoolField   bool   `yaml:"boolField"`
	}

	input := PrimitiveStruct{
		IntField:    42,
		StringField: "test",
		BoolField:   true,
	}

	result, err := StructToValueMap(input)
	require.NoError(t, err)

	assert.Equal(t, 42, result["intField"])
	assert.Equal(t, "test", result["stringField"])
	assert.Equal(t, true, result["boolField"])
}

func TestStructToValueMap_NestedStruct(t *testing.T) {
	type InnerStruct struct {
		Name  string `yaml:"name"`
		Value int    `yaml:"value"`
	}

	type OuterStruct struct {
		ID    int         `yaml:"id"`
		Inner InnerStruct `yaml:"inner"`
	}

	input := OuterStruct{
		ID: 1,
		Inner: InnerStruct{
			Name:  "nested",
			Value: 100,
		},
	}

	result, err := StructToValueMap(input)
	require.NoError(t, err)

	assert.Equal(t, 1, result["id"])

	inner, ok := result["inner"].(map[interface{}]interface{})
	require.True(t, ok, "inner should be a map")
	assert.Equal(t, "nested", inner["name"])
	assert.Equal(t, 100, inner["value"])
}

func TestStructToValueMap_MapField(t *testing.T) {
	type StructWithMap struct {
		Name       string            `yaml:"name"`
		Labels     map[string]string `yaml:"labels"`
		Attributes map[string]int    `yaml:"attributes"`
	}

	input := StructWithMap{
		Name: "test",
		Labels: map[string]string{
			"env":  "prod",
			"team": "platform",
		},
		Attributes: map[string]int{
			"replicas": 3,
			"timeout":  30,
		},
	}

	result, err := StructToValueMap(input)
	require.NoError(t, err)

	assert.Equal(t, "test", result["name"])

	labels, ok := result["labels"].(map[interface{}]interface{})
	require.True(t, ok, "labels should be a map")
	assert.Equal(t, "prod", labels["env"])
	assert.Equal(t, "platform", labels["team"])

	attributes, ok := result["attributes"].(map[interface{}]interface{})
	require.True(t, ok, "attributes should be a map")
	assert.Equal(t, 3, attributes["replicas"])
	assert.Equal(t, 30, attributes["timeout"])
}

func TestStructToValueMap_SliceField(t *testing.T) {
	type StructWithSlice struct {
		Name    string   `yaml:"name"`
		Tags    []string `yaml:"tags"`
		Numbers []int    `yaml:"numbers"`
	}

	input := StructWithSlice{
		Name:    "test",
		Tags:    []string{"kubernetes", "helm", "controller"},
		Numbers: []int{1, 2, 3, 4, 5},
	}

	result, err := StructToValueMap(input)
	require.NoError(t, err)

	assert.Equal(t, "test", result["name"])

	tags, ok := result["tags"].([]interface{})
	require.True(t, ok, "tags should be a slice")
	assert.Len(t, tags, 3)
	assert.Equal(t, "kubernetes", tags[0])
	assert.Equal(t, "helm", tags[1])
	assert.Equal(t, "controller", tags[2])

	numbers, ok := result["numbers"].([]interface{})
	require.True(t, ok, "numbers should be a slice")
	assert.Len(t, numbers, 5)
	assert.Equal(t, 1, numbers[0])
	assert.Equal(t, 5, numbers[4])
}

func TestStructToValueMap_SliceOfStructs(t *testing.T) {
	type Item struct {
		Key   string `yaml:"key"`
		Value int    `yaml:"value"`
	}

	type StructWithStructSlice struct {
		Name  string `yaml:"name"`
		Items []Item `yaml:"items"`
	}

	input := StructWithStructSlice{
		Name: "collection",
		Items: []Item{
			{Key: "first", Value: 1},
			{Key: "second", Value: 2},
			{Key: "third", Value: 3},
		},
	}

	result, err := StructToValueMap(input)
	require.NoError(t, err)

	assert.Equal(t, "collection", result["name"])

	items, ok := result["items"].([]interface{})
	require.True(t, ok, "items should be a slice")
	assert.Len(t, items, 3)

	firstItem, ok := items[0].(map[interface{}]interface{})
	require.True(t, ok, "first item should be a map")
	assert.Equal(t, "first", firstItem["key"])
	assert.Equal(t, 1, firstItem["value"])
}

func TestStructToValueMap_ComplexNested(t *testing.T) {
	type Resource struct {
		CPU    string `yaml:"cpu"`
		Memory string `yaml:"memory"`
	}

	type Container struct {
		Name      string            `yaml:"name"`
		Image     string            `yaml:"image"`
		Resources Resource          `yaml:"resources"`
		Env       map[string]string `yaml:"env"`
		Ports     []int             `yaml:"ports"`
	}

	type Deployment struct {
		Name       string            `yaml:"name"`
		Replicas   int               `yaml:"replicas"`
		Containers []Container       `yaml:"containers"`
		Labels     map[string]string `yaml:"labels"`
	}

	input := Deployment{
		Name:     "my-app",
		Replicas: 3,
		Containers: []Container{
			{
				Name:  "app",
				Image: "my-app:latest",
				Resources: Resource{
					CPU:    "100m",
					Memory: "128Mi",
				},
				Env: map[string]string{
					"LOG_LEVEL": "info",
					"PORT":      "8080",
				},
				Ports: []int{8080, 9090},
			},
		},
		Labels: map[string]string{
			"app": "my-app",
			"env": "prod",
		},
	}

	result, err := StructToValueMap(input)
	require.NoError(t, err)

	assert.Equal(t, "my-app", result["name"])
	assert.Equal(t, 3, result["replicas"])

	containers, ok := result["containers"].([]interface{})
	require.True(t, ok, "containers should be a slice")
	assert.Len(t, containers, 1)

	container, ok := containers[0].(map[interface{}]interface{})
	require.True(t, ok, "container should be a map")
	assert.Equal(t, "app", container["name"])
	assert.Equal(t, "my-app:latest", container["image"])

	resources, ok := container["resources"].(map[interface{}]interface{})
	require.True(t, ok, "resources should be a map")
	assert.Equal(t, "100m", resources["cpu"])
	assert.Equal(t, "128Mi", resources["memory"])

	env, ok := container["env"].(map[interface{}]interface{})
	require.True(t, ok, "env should be a map")
	assert.Equal(t, "info", env["LOG_LEVEL"])

	ports, ok := container["ports"].([]interface{})
	require.True(t, ok, "ports should be a slice")
	assert.Len(t, ports, 2)
	assert.Equal(t, 8080, ports[0])
}

func TestStructToValueMap_EmptyStruct(t *testing.T) {
	type EmptyStruct struct{}

	input := EmptyStruct{}

	result, err := StructToValueMap(input)
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestStructToValueMap_NilValues(t *testing.T) {
	type StructWithPointers struct {
		Name  string  `yaml:"name"`
		Value *int    `yaml:"value,omitempty"`
		Data  *string `yaml:"data,omitempty"`
	}

	input := StructWithPointers{
		Name:  "test",
		Value: nil,
		Data:  nil,
	}

	result, err := StructToValueMap(input)
	require.NoError(t, err)

	assert.Equal(t, "test", result["name"])
	// omitempty should exclude nil pointers
	_, hasValue := result["value"]
	_, hasData := result["data"]
	assert.False(t, hasValue)
	assert.False(t, hasData)
}

func TestStructToValueMap_ZeroValues(t *testing.T) {
	type StructWithZeros struct {
		IntField    int    `yaml:"intField"`
		StringField string `yaml:"stringField"`
		BoolField   bool   `yaml:"boolField"`
		SliceField  []int  `yaml:"sliceField"`
	}

	input := StructWithZeros{}

	result, err := StructToValueMap(input)
	require.NoError(t, err)

	assert.Equal(t, 0, result["intField"])
	assert.Equal(t, "", result["stringField"])
	assert.Equal(t, false, result["boolField"])
	// YAML marshals nil slice as empty slice
	sliceField, ok := result["sliceField"].([]interface{})
	require.True(t, ok)
	assert.Empty(t, sliceField)
}

func TestStructToValueMap_NumericTypes(t *testing.T) {
	type NumericStruct struct {
		Int8Field    int8    `yaml:"int8Field"`
		Int16Field   int16   `yaml:"int16Field"`
		Int32Field   int32   `yaml:"int32Field"`
		Int64Field   int64   `yaml:"int64Field"`
		Uint8Field   uint8   `yaml:"uint8Field"`
		Uint16Field  uint16  `yaml:"uint16Field"`
		Uint32Field  uint32  `yaml:"uint32Field"`
		Uint64Field  uint64  `yaml:"uint64Field"`
		Float32Field float32 `yaml:"float32Field"`
		Float64Field float64 `yaml:"float64Field"`
	}

	input := NumericStruct{
		Int8Field:    127,
		Int16Field:   32767,
		Int32Field:   2147483647,
		Int64Field:   9223372036854775807,
		Uint8Field:   255,
		Uint16Field:  65535,
		Uint32Field:  4294967295,
		Uint64Field:  18446744073709551615,
		Float32Field: 3.14,
		Float64Field: 2.718281828,
	}

	result, err := StructToValueMap(input)
	require.NoError(t, err)

	// YAML marshaling converts various numeric types to int (for integers) or float64 (for floats)
	// So we test values rather than exact types
	assert.Equal(t, 127, result["int8Field"])
	assert.Equal(t, 32767, result["int16Field"])
	assert.Equal(t, 2147483647, result["int32Field"])
	assert.Equal(t, 9223372036854775807, result["int64Field"])
	assert.Equal(t, 255, result["uint8Field"])
	assert.Equal(t, 65535, result["uint16Field"])
	assert.Equal(t, 4294967295, result["uint32Field"])
	assert.Equal(t, uint64(18446744073709551615), result["uint64Field"])
	assert.InDelta(t, 3.14, result["float32Field"], 0.01)
	assert.InDelta(t, 2.718281828, result["float64Field"], 0.0001)
}
