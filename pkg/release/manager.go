package release

import (
	"context"
	"path/filepath"

	tacokumogithubiov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/appconfig"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/helmutil"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/repoconnector"

	"github.com/go-logr/logr"
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
	repo, err := repoconnector.CloneApplicationRepository(
		ctx,
		m.connector,
		rel.Spec.Repo.URL,
		referenceName,
		rel.Spec.AppConfigPath)
	if err != nil {
		return err
	}

	values, err := m.constructReleaseValues(rel, &repo)
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
	repo *appconfig.Repository,
) (map[string]interface{}, error) {
	values := applicationchart.Values{
		Main: applicationchart.MainConfig{
			ApplicationName: rel.Name,
			Image:           repo.AppConfig.Build.Image,
			ReplicaCount:    1,
		},
	}
	return helmutil.StructToValueMap(values)
}
