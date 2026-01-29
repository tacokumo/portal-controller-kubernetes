package release

import (
	"context"

	tacokumogithubiov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"

	"github.com/go-logr/logr"
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
	rel *tacokumogithubiov1alpha1.Release,
) error {
	switch rel.Status.State {
	default:
		rel.Status.State = tacokumogithubiov1alpha1.ReleaseStateDeploying
	}

	if err := m.k8sClient.Status().Update(ctx, rel); err != nil {
		return m.handleError(ctx, rel, err)
	}
	return nil
}

func (m *Manager) handleError(
	ctx context.Context,
	rel *tacokumogithubiov1alpha1.Release,
	err error,
) error {
	// 引数のerrorは必ずnilではない
	rel.Status.State = tacokumogithubiov1alpha1.ApplicationStateError
	// errorだとしても､Statusの更新は必要
	if updateErr := m.k8sClient.Status().Update(ctx, rel); updateErr != nil {
		return updateErr
	}
	return err
}
