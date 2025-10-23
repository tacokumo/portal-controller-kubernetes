package application

import (
	"context"
	"path/filepath"

	tacokumoiov1alpha1 "tacokumo/portal-controller-kubernetes/api/v1alpha1"
	tacokumoapplication "tacokumo/portal-controller-kubernetes/charts/tacokumo-application"
	"tacokumo/portal-controller-kubernetes/pkg/appconfig"
	"tacokumo/portal-controller-kubernetes/pkg/helmutil"

	"go.yaml.in/yaml/v2"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/go-git/go-billy/v6/memfs"
	"github.com/go-git/go-git/v6"
	"github.com/go-git/go-git/v6/plumbing"
	"github.com/go-git/go-git/v6/storage/memory"
	"github.com/go-logr/logr"
	apispec "github.com/tacokumo/api-spec"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
			return err
		}
	case tacokumoiov1alpha1.ApplicationStateWaiting:
		if err := m.reconcileOnWaitingState(ctx, app); err != nil {
			return err
		}
	default:
		// すぐ遷移して終了
		app.Status.State = tacokumoiov1alpha1.ApplicationStateProvisioning
		return nil
	}

	if err := m.k8sClient.Status().Update(ctx, app); err != nil {
		return err
	}

	return nil
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
		if err := createOrUpdateObject(ctx, m.k8sClient, obj); err != nil {
			return err
		}
	}

	app.Status.State = tacokumoiov1alpha1.ApplicationStateWaiting
	return nil
}

func (m *Manager) reconcileOnWaitingState(
	_ context.Context,
	app *tacokumoiov1alpha1.Application,
) (err error) {
	// TODO: deploymentのPodがすべてReadyになっていることを確認する

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

func createOrUpdateObject(ctx context.Context, k8sClient client.Client, obj *unstructured.Unstructured) error {
	existingObj := &unstructured.Unstructured{}
	existingObj.SetGroupVersionKind(obj.GroupVersionKind())
	err := k8sClient.Get(ctx, client.ObjectKey{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}, existingObj)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		// Not found, create
		return k8sClient.Create(ctx, obj)
	}
	// Found, update
	obj.SetResourceVersion(existingObj.GetResourceVersion())
	return k8sClient.Update(ctx, obj)
}
