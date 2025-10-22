package helmutil

import (
	"bytes"
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	k8syaml "k8s.io/apimachinery/pkg/util/yaml"
)

// ParseManifestsToUnstructuredはYAML形式のマニフェストをUnstructuredオブジェクトのスライスに変換する
func ParseManifestsToUnstructured(manifestsYAML string) ([]*unstructured.Unstructured, error) {
	var objects []*unstructured.Unstructured

	decoder := k8syaml.NewYAMLOrJSONDecoder(bytes.NewBufferString(manifestsYAML), 4096)

	for {
		var obj unstructured.Unstructured
		err := decoder.Decode(&obj)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}

		if obj.Object != nil {
			objects = append(objects, &obj)
		}
	}

	return objects, nil
}
