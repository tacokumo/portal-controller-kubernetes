package application

import (
	"fmt"
	tacokumoiov1alpha1 "tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"testing"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestManager_reconcileOnProvisioningState(t *testing.T) {
	tests := []struct {
		name                  string
		app                   *tacokumoiov1alpha1.Application
		expectedState         string
		afterReconcileCheckFn []func(app *tacokumoiov1alpha1.Application, k8sClient client.Client) error
	}{
		{
			name: "basic",
			app: &tacokumoiov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "basic",
					Name:      "basic",
				},
				Spec: tacokumoiov1alpha1.ApplicationSpec{
					AppConfigPath: "portal-controller-kubernetes/basic/appconfig.yaml",
					Repo: tacokumoiov1alpha1.RepositoryRef{
						URL: "https://github.com/tacokumo/git-fixtures.git",
						Ref: "main",
					},
				},
			},
			afterReconcileCheckFn: []func(app *tacokumoiov1alpha1.Application, k8sClient client.Client) error{
				func(app *tacokumoiov1alpha1.Application, k8sClient client.Client) error {
					dep := appsv1.Deployment{}
					err := k8sClient.Get(t.Context(), types.NamespacedName{Namespace: app.Namespace, Name: "basic"}, &dep)
					if err != nil {
						return err
					}

					if dep.Spec.Template.Spec.Containers[0].Image != "ghcr.io/tacokumo/tacokumo-bot:main" {
						return fmt.Errorf("unexpected image: %s", dep.Spec.Template.Spec.Containers[0].Image)
					}

					return nil
				},
			},
			expectedState: tacokumoiov1alpha1.ApplicationStateWaiting,
		},
		{
			name: "env secret",
			app: &tacokumoiov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "basic",
					Name:      "basic",
				},
				Spec: tacokumoiov1alpha1.ApplicationSpec{
					AppConfigPath: "portal-controller-kubernetes/basic/appconfig.yaml",
					Repo: tacokumoiov1alpha1.RepositoryRef{
						URL: "https://github.com/tacokumo/git-fixtures.git",
						Ref: "main",
					},
					EnvSecretName: ptr.To("basic-env-secret"),
				},
			},
			afterReconcileCheckFn: []func(app *tacokumoiov1alpha1.Application, k8sClient client.Client) error{
				func(app *tacokumoiov1alpha1.Application, k8sClient client.Client) error {
					dep := appsv1.Deployment{}
					err := k8sClient.Get(t.Context(), types.NamespacedName{Namespace: app.Namespace, Name: "basic"}, &dep)
					if err != nil {
						return err
					}

					if len(dep.Spec.Template.Spec.Containers[0].EnvFrom) != 1 {
						return fmt.Errorf("unexpected EnvFrom length: %d", len(dep.Spec.Template.Spec.Containers[0].EnvFrom))
					}
					if dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef == nil {
						return fmt.Errorf("unexpected EnvFrom SecretRef is nil")
					}
					if dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.Name != "basic-env-secret" {
						secretName := dep.Spec.Template.Spec.Containers[0].EnvFrom[0].SecretRef.Name
						return fmt.Errorf("unexpected EnvFrom SecretRef Name: %s", secretName)
					}
					return nil
				},
			},
			expectedState: tacokumoiov1alpha1.ApplicationStateWaiting,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(scheme)
			assert.NoError(t, err)
			err = tacokumoiov1alpha1.AddToScheme(scheme)
			assert.NoError(t, err)

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			manager := &Manager{
				k8sClient: k8sClient,
				workdir:   "../../",
			}

			err = manager.reconcileOnProvisioningState(t.Context(), tt.app)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedState, tt.app.Status.State)

			for _, fn := range tt.afterReconcileCheckFn {
				err = fn(tt.app, k8sClient)
				assert.NoError(t, err)
			}
		})
	}
}

func TestManager_cloneApplicationRepository(t *testing.T) {
	app := &tacokumoiov1alpha1.Application{
		Spec: tacokumoiov1alpha1.ApplicationSpec{
			AppConfigPath: "portal-controller-kubernetes/basic/appconfig.yaml",
			Repo: tacokumoiov1alpha1.RepositoryRef{
				URL: "https://github.com/tacokumo/git-fixtures.git",
				Ref: "main",
			},
		},
	}

	manager := &Manager{
		workdir: "../../",
	}
	repo, err := manager.cloneApplicationRepository(t.Context(), app)
	assert.NoError(t, err)

	assert.Equal(t, "basic", repo.AppConfig.AppName)
	assert.Equal(t, "ghcr.io/tacokumo/tacokumo-bot:main", repo.AppConfig.Build.Image)
}
