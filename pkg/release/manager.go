package release

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/samber/lo"
	tacokumogithubiov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/helmutil"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/repoconnector"

	"github.com/go-logr/logr"
	appconfig "github.com/tacokumo/appconfig"
	applicationchart "github.com/tacokumo/helm-charts/charts/tacokumo-application"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Manager struct {
	logger    logr.Logger
	k8sClient client.Client
	connector repoconnector.GitRepositoryConnector
	workdir   string
}

func NewManager(
	logger logr.Logger,
	k8sClient client.Client,
	workdir string,
) *Manager {
	return &Manager{
		logger:    logger,
		k8sClient: k8sClient,
		workdir:   workdir,
		connector: repoconnector.NewDefaultConnector(),
	}
}

// WithConnector は Manager に GitRepositoryConnector を設定する
// テスト用に公開されている
func (m *Manager) WithConnector(connector repoconnector.GitRepositoryConnector) *Manager {
	m.connector = connector
	return m
}

func (m *Manager) Reconcile(
	ctx context.Context,
	rel *tacokumogithubiov1alpha1.Release,
) error {
	switch rel.Status.State {
	case tacokumogithubiov1alpha1.ReleaseStateDeploying:
		if err := m.reconcileOnDeployingState(ctx, rel); err != nil {
			return m.handleError(ctx, rel, err)
		}
	default:
		rel.Status.State = tacokumogithubiov1alpha1.ReleaseStateDeploying
	}

	if err := m.k8sClient.Status().Update(ctx, rel); err != nil {
		return m.handleError(ctx, rel, err)
	}
	return nil
}

func (m *Manager) reconcileOnDeployingState(
	ctx context.Context,
	rel *tacokumogithubiov1alpha1.Release,
) error {
	referenceName := rel.Spec.AppConfigBranch
	if referenceName == "" {
		referenceName = *rel.Spec.Commit
	}
	appCfg, err := repoconnector.CloneApplicationRepository(
		ctx,
		m.connector,
		rel.Spec.Repo.URL,
		referenceName,
		rel.Spec.AppConfigPath)
	if err != nil {
		return err
	}

	values, err := m.constructReleaseValues(rel, &appCfg)
	if err != nil {
		return err
	}

	chartPath := filepath.Join(m.workdir, "helm-charts", "charts", "tacokumo-application")

	manifests, err := helmutil.RenderChart(chartPath, rel.Name, rel.Namespace, values)
	if err != nil {
		return err
	}

	objects, err := helmutil.ParseManifestsToUnstructured(manifests)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		obj.SetNamespace(rel.Namespace)
		if err := helmutil.CreateOrUpdateObject(ctx, m.k8sClient, obj); err != nil {
			return err
		}
	}

	rel.Status.State = tacokumogithubiov1alpha1.ReleaseStateDeployed
	return nil
}

func (m *Manager) handleError(
	ctx context.Context,
	rel *tacokumogithubiov1alpha1.Release,
	err error,
) error {
	// 引数のerrorは必ずnilではない
	rel.Status.State = tacokumogithubiov1alpha1.ReleaseStateFailed
	// errorだとしても､Statusの更新は必要
	if updateErr := m.k8sClient.Status().Update(ctx, rel); updateErr != nil {
		return updateErr
	}
	return err
}

func (m *Manager) constructReleaseValues(
	rel *tacokumogithubiov1alpha1.Release,
	appCfg *appconfig.AppConfig,
) (map[string]interface{}, error) {
	hpa := constructHPAValues(appCfg)
	svc := constructServiceValues(appCfg)
	resource := constructResourceValues(appCfg)

	// TODO: probe, env, annotation

	values := applicationchart.Values{
		Main: applicationchart.MainConfig{
			ApplicationName: rel.Name,
			Image:           appCfg.Build.Image,
			Service:         svc,
			HPA:             hpa,
			Resources:       resource,
		},
	}
	return helmutil.StructToValueMap(values)
}

func constructHPAValues(
	appCfg *appconfig.AppConfig,
) applicationchart.HPAConfig {
	// TODO: メトリクスの指定はまだできない
	if appCfg.Service.Scale == nil {
		return applicationchart.HPAConfig{
			MinReplicas:                       1,
			MaxReplicas:                       1,
			TargetMemoryUtilizationPercentage: 50,
		}
	}
	return applicationchart.HPAConfig{
		MinReplicas:                       appCfg.Service.Scale.Min,
		MaxReplicas:                       appCfg.Service.Scale.Max,
		TargetMemoryUtilizationPercentage: 50, // TODO: 可変にする
	}
}

func constructServiceValues(
	appCfg *appconfig.AppConfig,
) applicationchart.ServiceConfig {
	// TODO: HTTP以外もサポートする
	if len(appCfg.Service.HTTP) == 0 {
		return applicationchart.ServiceConfig{
			Enabled: false,
		}
	}

	ports := lo.Map(appCfg.Service.HTTP, func(c appconfig.ServiceHTTPConfig, _ int) applicationchart.ServicePortConfig {
		return applicationchart.ServicePortConfig{
			Name:       fmt.Sprintf("http-%d", c.TargetPort),
			Port:       c.TargetPort, // NOTE: 本当にtagretPortで良いのか？
			TargetPort: c.TargetPort,
			Protocol:   "TCP",
		}
	})

	return applicationchart.ServiceConfig{
		Enabled: true,
		Type:    "ClusterIP",
		Ports:   ports,
	}
}

func constructResourceValues(
	appCfg *appconfig.AppConfig,
) applicationchart.ResourceConfig {
	// TODO: flavor
	if appCfg.Service.MachineConfig == nil {
		return applicationchart.ResourceConfig{
			Limits: applicationchart.ResourceSpec{
				CPU:    "100m",
				Memory: "128Mi",
			},
		}
	}
	return applicationchart.ResourceConfig{
		// TODO: リソース指定をサポートする
		Limits: applicationchart.ResourceSpec{
			CPU:    appCfg.Service.MachineConfig.CPU,
			Memory: appCfg.Service.MachineConfig.Memory,
		},
	}
}
