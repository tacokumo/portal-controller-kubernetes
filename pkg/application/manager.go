package application

import (
	"context"
	"fmt"

	tacokumogithubiov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/repoconnector"

	"github.com/go-logr/logr"
	appconfig "github.com/tacokumo/appconfig"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	defaultAppConfigBranch = "main"
)

type Manager struct {
	logger    logr.Logger
	k8sClient client.Client
	connector repoconnector.GitRepositoryConnector
}

func NewManager(
	logger logr.Logger,
	k8sClient client.Client) *Manager {
	return &Manager{
		logger:    logger,
		k8sClient: k8sClient,
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
	app *tacokumogithubiov1alpha1.Application,
) error {
	switch app.Status.State {
	case tacokumogithubiov1alpha1.ApplicationStateProvisioning:
		if err := m.reconcileOnProvisioningState(ctx, app); err != nil {
			return m.handleError(ctx, app, err)
		}
	case tacokumogithubiov1alpha1.ApplicationStateWaiting:
		if err := m.reconcileOnWaitingState(ctx, app); err != nil {
			return m.handleError(ctx, app, err)
		}
	case tacokumogithubiov1alpha1.ApplicationStateRunning:
		// TODO: 差分を検知したらProvisioningに戻す
	case tacokumogithubiov1alpha1.ApplicationStateError:
		// TODO: 差分を検知したらProvisioningに戻す
	default:
		app.Status.State = tacokumogithubiov1alpha1.ApplicationStateProvisioning
	}

	if err := m.k8sClient.Status().Update(ctx, app); err != nil {
		return m.handleError(ctx, app, err)
	}
	return nil
}

func (m *Manager) handleError(
	ctx context.Context,
	app *tacokumogithubiov1alpha1.Application,
	err error,
) error {
	// 引数のerrorは必ずnilではない
	app.Status.State = tacokumogithubiov1alpha1.ApplicationStateError
	// errorだとしても､Statusの更新は必要
	if updateErr := m.k8sClient.Status().Update(ctx, app); updateErr != nil {
		return updateErr
	}
	return err
}

func (m *Manager) reconcileOnProvisioningState(
	ctx context.Context,
	app *tacokumogithubiov1alpha1.Application,
) (err error) {
	referenceName := app.Spec.ReleaseTemplate.AppConfigBranch
	if referenceName == "" {
		referenceName = defaultAppConfigBranch
	}
	appCfg, err := repoconnector.CloneApplicationRepository(
		ctx,
		m.connector,
		app.Spec.ReleaseTemplate.Repo.URL,
		referenceName,
		app.Spec.ReleaseTemplate.AppConfigPath)
	if err != nil {
		return err
	}

	if len(appCfg.Stages) == 0 {
		appCfg.Stages = m.setDefaultStages()
	}

	app.Status.Releases = make([]corev1.ObjectReference, 0, len(appCfg.Stages))
	for _, stage := range appCfg.Stages {
		rel := tacokumogithubiov1alpha1.Release{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: app.Namespace,
				Name:      fmt.Sprintf("%s-%s", app.Name, stage.Name),
			},
		}

		if _, err := controllerutil.CreateOrUpdate(ctx, m.k8sClient, &rel, func() error {
			rel.Spec = app.Spec.ReleaseTemplate
			return nil
		}); err != nil {
			return err
		}
		app.Status.Releases = append(app.Status.Releases, corev1.ObjectReference{
			Kind:      rel.Kind,
			Namespace: rel.Namespace,
			Name:      rel.Name,
			UID:       rel.UID,
		})
	}

	app.Status.State = tacokumogithubiov1alpha1.ApplicationStateWaiting
	return nil
}

func (m *Manager) reconcileOnWaitingState(
	ctx context.Context,
	app *tacokumogithubiov1alpha1.Application,
) (err error) {
	for _, relRef := range app.Status.Releases {
		rel := &tacokumogithubiov1alpha1.Release{}
		if err := m.k8sClient.Get(ctx, client.ObjectKey{
			Namespace: relRef.Namespace,
			Name:      relRef.Name,
		}, rel); err != nil {
			return err
		}
		if rel.Status.State != tacokumogithubiov1alpha1.ReleaseStateDeployed {
			m.logger.Info("waiting for all Releases to be in Deployed state",
				"release", fmt.Sprintf("%s/%s", rel.Namespace, rel.Name),
				"state", rel.Status.State,
			)
			return nil
		}
	}
	app.Status.State = tacokumogithubiov1alpha1.ApplicationStateRunning
	return nil
}

// setDefaultStages は､AppConfigにStagesが定義されていない場合のデフォルト値を返す
func (m *Manager) setDefaultStages() []appconfig.StageConfig {
	return []appconfig.StageConfig{
		{
			Name: "production",
			Policy: appconfig.StagePolicyConfig{
				Type: "branch",
				Branch: &appconfig.BranchConfig{
					Name: "main",
				},
			},
		},
	}
}
