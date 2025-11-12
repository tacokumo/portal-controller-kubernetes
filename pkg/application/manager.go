package application

import (
	"context"
	"path/filepath"

	tacokumoiov1alpha1 "tacokumo/portal-controller-kubernetes/api/v1alpha1"
	tacokumoapplication "tacokumo/portal-controller-kubernetes/charts/tacokumo-application"
	"tacokumo/portal-controller-kubernetes/pkg/appconfig"
	"tacokumo/portal-controller-kubernetes/pkg/helmutil"
	"tacokumo/portal-controller-kubernetes/pkg/requeue"

	"go.yaml.in/yaml/v2"

	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/storage/memory"
	"github.com/go-logr/logr"
	apispec "github.com/tacokumo/api-spec"
	"sigs.k8s.io/controller-runtime/pkg/client"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type Manager struct {
	logger    logr.Logger
	k8sClient client.Client
	workdir   string
}

func NewManager(
	logger logr.Logger,
	k8sClient client.Client,
	workdir string) *Manager {
	return &Manager{
		logger:    logger,
		k8sClient: k8sClient,
		workdir:   workdir,
	}
}

func (m *Manager) Reconcile(
	ctx context.Context,
	app *tacokumoiov1alpha1.Application,
) error {
	switch app.Status.State {
	case tacokumoiov1alpha1.ApplicationStateProvisioning:
		if err := m.reconcileOnProvisioningState(ctx, app); err != nil {
			return m.handleError(ctx, app, err)
		}
	case tacokumoiov1alpha1.ApplicationStateWaiting:
		if err := m.reconcileOnWaitingState(ctx, app); err != nil {
			return m.handleError(ctx, app, err)
		}
	case tacokumoiov1alpha1.ApplicationStateRunning:
		// TODO: 差分を検知したらProvisioningに戻す
	case tacokumoiov1alpha1.ApplicationStateError:
		// TODO: 差分を検知したらProvisioningに戻す
	default:
		app.Status.State = tacokumoiov1alpha1.ApplicationStateProvisioning
	}

	if err := m.k8sClient.Status().Update(ctx, app); err != nil {
		return m.handleError(ctx, app, err)
	}
	return nil
}

func (m *Manager) handleError(
	ctx context.Context,
	app *tacokumoiov1alpha1.Application,
	err error,
) error {
	// 引数のerrorは必ずnilではない
	app.Status.State = tacokumoiov1alpha1.ApplicationStateError
	// errorだとしても､Statusの更新は必要
	if updateErr := m.k8sClient.Status().Update(ctx, app); updateErr != nil {
		return updateErr
	}
	if requeue.IsRequeueError(err) {
		return nil
	}
	return err
}

func (m *Manager) reconcileOnProvisioningState(
	ctx context.Context,
	app *tacokumoiov1alpha1.Application,
) (err error) {
	repo, err := m.cloneApplicationRepository(ctx, app)
	if err != nil {
		return err
	}

	values := m.constructValues(repo, app)

	chartPath := filepath.Join(m.workdir, "charts", "tacokumo-application")

	valueMap, err := helmutil.StructToValueMap(values)
	if err != nil {
		return err
	}
	manifests, err := helmutil.RenderChart(chartPath, app.Name, app.Namespace, valueMap)
	if err != nil {
		return err
	}

	objects, err := helmutil.ParseManifestsToUnstructured(manifests)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		obj.SetNamespace(app.Namespace)
		if err := helmutil.CreateOrUpdateObject(ctx, m.k8sClient, obj); err != nil {
			return err
		}
	}

	app.Status.State = tacokumoiov1alpha1.ApplicationStateWaiting
	return nil
}

func (m *Manager) reconcileOnWaitingState(
	ctx context.Context,
	app *tacokumoiov1alpha1.Application,
) (err error) {
	deploymentList := appsv1.DeploymentList{}
	err = m.k8sClient.List(ctx, &deploymentList, client.InNamespace(app.Namespace), client.MatchingLabels{
		tacokumoiov1alpha1.ManagedByLabelKey: "portal-controller",
	})
	if err != nil {
		return err
	}

	for _, deployment := range deploymentList.Items {
		app.Status.Deployments = append(app.Status.Deployments, tacokumoiov1alpha1.NamespacedName{
			Namespace: deployment.GetNamespace(),
			Name:      deployment.GetName(),
		})
	}

	podList := corev1.PodList{}
	err = m.k8sClient.List(ctx, &podList, client.InNamespace(app.Namespace), client.MatchingLabels{
		tacokumoiov1alpha1.ManagedByLabelKey: "portal-controller",
	})
	if err != nil {
		return err
	}

	allReady := true
	for _, pod := range podList.Items {
		if pod.Status.Phase != corev1.PodRunning {
			allReady = false
		}

		app.Status.Pods = append(app.Status.Pods, tacokumoiov1alpha1.PodReference{
			NamespacedName: tacokumoiov1alpha1.NamespacedName{
				Namespace: pod.GetNamespace(),
				Name:      pod.GetName(),
			},
			Ready: pod.Status.Phase == corev1.PodRunning,
		})
	}

	if !allReady {
		return requeue.NewError("some pods are not ready yet")
	}

	// TODO: healthcheckを実行もしくは監視し、成功していることを確認する
	app.Status.State = tacokumoiov1alpha1.ApplicationStateRunning
	return nil
}

func (m *Manager) cloneApplicationRepository(
	ctx context.Context,
	app *tacokumoiov1alpha1.Application,
) (repo appconfig.Repository, err error) {
	fs := memfs.New()
	storer := memory.NewStorage()
	gitRepo, err := git.CloneContext(ctx, storer, fs, &git.CloneOptions{
		URL:           app.Spec.Repo.URL,
		ReferenceName: plumbing.NewBranchReferenceName(app.Spec.Repo.Ref),
	})
	if err != nil {
		return appconfig.Repository{}, err
	}
	wt, err := gitRepo.Worktree()
	if err != nil {
		return appconfig.Repository{}, err
	}
	f, err := wt.Filesystem.Open(app.Spec.AppConfigPath)
	if err != nil {
		return appconfig.Repository{}, err
	}
	defer func() {
		err = f.Close()
	}()

	appConfig := apispec.AppConfig{}
	if err := yaml.NewDecoder(f).Decode(&appConfig); err != nil {
		return appconfig.Repository{}, err
	}

	return appconfig.Repository{
		AppConfig: appConfig,
	}, nil
}

func (m *Manager) constructValues(
	repo appconfig.Repository,
	_ *tacokumoiov1alpha1.Application,
) tacokumoapplication.Values {
	values := tacokumoapplication.Values{
		Main: tacokumoapplication.MainApplicationValues{
			ApplicationName: repo.AppConfig.AppName,
			ReplicaCount:    1,
			Image:           repo.AppConfig.Build.Image,
		},
	}
	return values
}
