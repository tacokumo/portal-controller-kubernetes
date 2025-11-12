package portal

import (
	"context"
	"path/filepath"

	tacokumoiov1alpha1 "tacokumo/portal-controller-kubernetes/api/v1alpha1"
	tacokumoportal "tacokumo/portal-controller-kubernetes/charts/tacokumo-portal"
	"tacokumo/portal-controller-kubernetes/pkg/helmutil"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
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
	p *tacokumoiov1alpha1.Portal,
) error {

	switch p.Status.State {
	case tacokumoiov1alpha1.PortalStateProvisioning:
		if err := m.reconcileOnProvisioningState(ctx, p); err != nil {
			return m.handleError(ctx, p, err)
		}
	case tacokumoiov1alpha1.PortalStateWaiting:
		if err := m.reconcileOnWaitingState(ctx, p); err != nil {
			return m.handleError(ctx, p, err)
		}
	case tacokumoiov1alpha1.PortalStateRunning:
		// TODO: 差分を検知したらProvisioningに戻す
	case tacokumoiov1alpha1.PortalStateError:
		// TODO: 差分を検知したらProvisioningに戻す
	default:
		p.Status.State = tacokumoiov1alpha1.PortalStateProvisioning
	}

	if err := m.k8sClient.Status().Update(ctx, p); err != nil {
		return m.handleError(ctx, p, err)
	}
	return nil
}

func (m *Manager) reconcileOnProvisioningState(
	ctx context.Context,
	p *tacokumoiov1alpha1.Portal,
) error {
	ns := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: p.Name,
		},
	}

	_, err := controllerutil.CreateOrUpdate(ctx, m.k8sClient, &ns, func() error {
		// namespaceは特に更新する項目はない
		return nil
	})
	if err != nil {
		return err
	}

	if err := m.k8sClient.Get(ctx, types.NamespacedName{Name: p.Name}, &ns); err != nil {
		// 作成されるまで待つ、コントローラーが自動的に再キューイングする
		return nil
	}

	values := m.constructValues(p)

	chartPath := filepath.Join(m.workdir, "charts", "tacokumo-portal")

	valueMap, err := helmutil.StructToValueMap(values)
	if err != nil {
		return err
	}
	manifests, err := helmutil.RenderChart(chartPath, p.Name, p.Name, valueMap)
	if err != nil {
		return err
	}

	objects, err := helmutil.ParseManifestsToUnstructured(manifests)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		obj.SetNamespace(p.Name)
		if err := helmutil.CreateOrUpdateObject(ctx, m.k8sClient, obj); err != nil {
			return err
		}
	}

	p.Status.State = tacokumoiov1alpha1.PortalStateWaiting
	return nil
}

func (m *Manager) reconcileOnWaitingState(
	ctx context.Context,
	p *tacokumoiov1alpha1.Portal,
) error {
	podList := corev1.PodList{}
	err := m.k8sClient.List(ctx, &podList, client.InNamespace(p.Name), client.MatchingLabels{
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
	}

	if !allReady {
		// pods are not ready yet, but the controller will requeue automatically
		return nil
	}

	// TODO: healthcheckを実行もしくは監視し、成功していることを確認する
	p.Status.State = tacokumoiov1alpha1.PortalStateRunning
	return nil
}

func (m *Manager) handleError(
	ctx context.Context,
	p *tacokumoiov1alpha1.Portal,
	err error,
) error {
	// 引数のerrorは必ずnilではない
	p.Status.State = tacokumoiov1alpha1.PortalStateError
	// errorだとしても､Statusの更新は必要
	if updateErr := m.k8sClient.Status().Update(ctx, p); updateErr != nil {
		return updateErr
	}
	return err
}

func (m *Manager) constructValues(
	p *tacokumoiov1alpha1.Portal,
) tacokumoportal.Values {
	values := tacokumoportal.Values{
		Namespace:  p.Name,
		NamePrefix: p.Name,
	}
	return values
}
