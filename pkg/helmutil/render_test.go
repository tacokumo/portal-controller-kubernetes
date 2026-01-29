package helmutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testChartPath = "testdata/test-chart"

// TestRenderChartは、RenderChart関数が正常に動作することを確認します。
// チャートの内容は頻繁に変更されるため、レンダリングされたマニフェストの具体的な内容はテストしません。
// エラーが発生せず、何かしらのマニフェストが生成されることのみを確認します。
func TestRenderChart(t *testing.T) {
	tests := []struct {
		name        string
		chartPath   string
		releaseName string
		namespace   string
		values      map[string]interface{}
		wantErr     bool
	}{
		{
			name:        "render with default values",
			chartPath:   testChartPath,
			releaseName: "test-release",
			namespace:   "default",
			values:      nil,
			wantErr:     false,
		},
		{
			name:        "render with custom values",
			chartPath:   testChartPath,
			releaseName: "custom-release",
			namespace:   "test-namespace",
			values: map[string]interface{}{
				"main": map[string]interface{}{
					"applicationName": "my-app",
					"replicaCount":    3,
					"image": map[string]interface{}{
						"repository": "myregistry/myapp",
						"tag":        "v1.0.0",
						"pullPolicy": "Always",
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "render with imagePullSecrets",
			chartPath:   testChartPath,
			releaseName: "secret-release",
			namespace:   "default",
			values: map[string]interface{}{
				"main": map[string]interface{}{
					"applicationName": "secret-app",
					"replicaCount":    1,
					"image": map[string]interface{}{
						"repository": "private/app",
						"tag":        "latest",
					},
					"imagePullSecrets": []map[string]interface{}{
						{"name": "regcred"},
					},
				},
			},
			wantErr: false,
		},
		{
			name:        "invalid chart path",
			chartPath:   "/nonexistent/chart/path",
			releaseName: "test-release",
			namespace:   "default",
			values:      nil,
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manifest, err := RenderChart(tt.chartPath, tt.releaseName, tt.namespace, tt.values)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			// 何かしらのマニフェストが生成されていることを確認
			assert.NotEmpty(t, manifest)
		})
	}
}

// TestRenderChart_ParseableOutputは、レンダリングされたマニフェストがパース可能であることを確認します。
// これにより、Helmのレンダリング結果が有効なKubernetesマニフェストであることを保証します。
func TestRenderChart_ParseableOutput(t *testing.T) {
	manifest, err := RenderChart(
		testChartPath,
		"parse-test",
		"default",
		nil,
	)

	require.NoError(t, err)

	// レンダリングされたマニフェストがパース可能であることを確認
	objects, err := ParseManifestsToUnstructured(manifest)
	require.NoError(t, err)
	// 少なくとも1つ以上のKubernetesオブジェクトが生成されていることを確認
	assert.NotEmpty(t, objects)
}
