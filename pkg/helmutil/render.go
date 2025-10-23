package helmutil

import (
	"strings"

	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// RenderChartはHelmチャートをレンダリングし、マニフェストの文字列を返す
// install actionを使わない理由は､client.Clientを使えるようにして､testableにするため
func RenderChart(chartPath, releaseName, namespace string, values map[string]interface{}) (string, error) {
	chart, err := loader.Load(chartPath)
	if err != nil {
		return "", err
	}

	chartValues := chart.Values
	if chartValues == nil {
		chartValues = make(map[string]interface{})
	}
	chartValues = chartutil.CoalesceTables(values, chartValues)

	options := chartutil.ReleaseOptions{
		Name:      releaseName,
		Namespace: namespace,
		IsInstall: true,
	}

	valuesToRender, err := chartutil.ToRenderValues(chart, chartValues, options, nil)
	if err != nil {
		return "", err
	}

	files, err := engine.Render(chart, valuesToRender)
	if err != nil {
		return "", err
	}

	var manifests string
	for name, content := range files {
		if !strings.HasSuffix(name, ".txt") && !strings.Contains(name, "/templates/_") {
			content = strings.TrimSpace(content)
			if content != "" {
				manifests += content + "\n---\n"
			}
		}
	}

	return manifests, nil
}
