package helmutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

func TestParseManifestsToUnstructured(t *testing.T) {
	tests := []struct {
		name          string
		manifestsYAML string
		wantCount     int
		wantErr       bool
		validate      func(t *testing.T, objects []*unstructured.Unstructured)
	}{
		{
			name: "single valid manifest",
			manifestsYAML: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test-config
  namespace: default
data:
  key: value`,
			wantCount: 1,
			wantErr:   false,
			validate: func(t *testing.T, objects []*unstructured.Unstructured) {
				assert.Equal(t, "v1", objects[0].GetAPIVersion())
				assert.Equal(t, "ConfigMap", objects[0].GetKind())
				assert.Equal(t, "test-config", objects[0].GetName())
				assert.Equal(t, "default", objects[0].GetNamespace())
			},
		},
		{
			name: "multiple manifests separated by ---",
			manifestsYAML: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
apiVersion: v1
kind: Service
metadata:
  name: service1
spec:
  ports:
  - port: 80`,
			wantCount: 2,
			wantErr:   false,
			validate: func(t *testing.T, objects []*unstructured.Unstructured) {
				assert.Equal(t, "ConfigMap", objects[0].GetKind())
				assert.Equal(t, "config1", objects[0].GetName())
				assert.Equal(t, "Service", objects[1].GetKind())
				assert.Equal(t, "service1", objects[1].GetName())
			},
		},
		{
			name:          "empty manifest",
			manifestsYAML: "",
			wantCount:     0,
			wantErr:       false,
		},
		{
			name:          "whitespace only",
			manifestsYAML: "   \n  \n  ",
			wantCount:     0,
			wantErr:       false,
		},
		{
			name: "manifest with null document",
			manifestsYAML: `---
null
---
apiVersion: v1
kind: Pod
metadata:
  name: test-pod`,
			wantCount: 1,
			wantErr:   false,
			validate: func(t *testing.T, objects []*unstructured.Unstructured) {
				assert.Equal(t, "Pod", objects[0].GetKind())
				assert.Equal(t, "test-pod", objects[0].GetName())
			},
		},
		{
			name: "invalid YAML",
			manifestsYAML: `apiVersion: v1
kind: ConfigMap
metadata:
  name: test
  invalid yaml here: [[[`,
			wantErr: true,
		},
		{
			name: "complex manifest with nested fields",
			manifestsYAML: `apiVersion: apps/v1
kind: Deployment
metadata:
  name: test-deployment
  labels:
    app: test
spec:
  replicas: 3
  selector:
    matchLabels:
      app: test
  template:
    metadata:
      labels:
        app: test
    spec:
      containers:
      - name: nginx
        image: nginx:latest
        ports:
        - containerPort: 80`,
			wantCount: 1,
			wantErr:   false,
			validate: func(t *testing.T, objects []*unstructured.Unstructured) {
				assert.Equal(t, "apps/v1", objects[0].GetAPIVersion())
				assert.Equal(t, "Deployment", objects[0].GetKind())
				assert.Equal(t, "test-deployment", objects[0].GetName())

				spec, found, err := unstructured.NestedInt64(objects[0].Object, "spec", "replicas")
				require.NoError(t, err)
				require.True(t, found)
				assert.Equal(t, int64(3), spec)
			},
		},
		{
			name: "manifest with empty documents between valid ones",
			manifestsYAML: `apiVersion: v1
kind: ConfigMap
metadata:
  name: config1
---
---
apiVersion: v1
kind: Secret
metadata:
  name: secret1
---`,
			wantCount: 2,
			wantErr:   false,
			validate: func(t *testing.T, objects []*unstructured.Unstructured) {
				assert.Equal(t, "ConfigMap", objects[0].GetKind())
				assert.Equal(t, "Secret", objects[1].GetKind())
			},
		},
		{
			name: "JSON format manifest",
			manifestsYAML: `{
  "apiVersion": "v1",
  "kind": "Service",
  "metadata": {
    "name": "test-service"
  },
  "spec": {
    "ports": [
      {
        "port": 8080
      }
    ]
  }
}`,
			wantCount: 1,
			wantErr:   false,
			validate: func(t *testing.T, objects []*unstructured.Unstructured) {
				assert.Equal(t, "Service", objects[0].GetKind())
				assert.Equal(t, "test-service", objects[0].GetName())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			objects, err := ParseManifestsToUnstructured(tt.manifestsYAML)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, objects, tt.wantCount)

			if tt.validate != nil {
				tt.validate(t, objects)
			}
		})
	}
}
