/*
MIT License

Copyright (c) 2025 tacokumo

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

package controller

import (
	"context"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	tacokumogithubiov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/release"
)

// ReleaseReconciler reconciles a Release object
type ReleaseReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=tacokumo.github.io,resources=releases,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=tacokumo.github.io,resources=releases/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=tacokumo.github.io,resources=releases/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Release object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.23.1/pkg/reconcile
func (r *ReleaseReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := logf.FromContext(ctx)
	key := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      req.Name,
	}
	rel := tacokumogithubiov1alpha1.Release{}
	if err := r.Get(ctx, key, &rel); err != nil {
		if apierrors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return ctrl.Result{}, nil
		}
	}

	manager := release.NewManager(logger, r.Client, ".")

	if err := manager.Reconcile(ctx, &rel); err != nil {
		// TODO: いつもrequeueするべきかどうかを考える
		logger.Error(err, "failed to reconcile with manager")
		return ctrl.Result{RequeueAfter: time.Second * 2}, nil
	}

	return ctrl.Result{RequeueAfter: time.Second * 2}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ReleaseReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&tacokumogithubiov1alpha1.Release{}).
		Named("release").
		Complete(r)
}
