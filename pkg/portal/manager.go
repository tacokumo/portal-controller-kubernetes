package portal

import (
	"context"

	"github.com/cockroachdb/errors"

	tacokumoiov1alpha1 "tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"tacokumo/portal-controller-kubernetes/pkg/requeue"

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
		// 作成されるまで待つ
		return requeue.NewError("waiting for Namespace to be created")
	}

	// TODO: portal helm chartをnamespaceにinstallする処理を追加

	p.Status.State = tacokumoiov1alpha1.PortalStateWaiting
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
	if errors.As(err, &requeue.Error{}) {
		return nil
	}
	return err
}
