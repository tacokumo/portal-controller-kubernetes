package application

import (
	tacokumoiov1alpha1 "tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"testing"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestManager_reconcileOnProvisioningState(t *testing.T) {
	tests := []struct {
		name          string
		app           *tacokumoiov1alpha1.Application
		expectedState string
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

			dep := appsv1.Deployment{}
			err = k8sClient.Get(t.Context(), types.NamespacedName{Namespace: tt.app.Namespace, Name: "basic"}, &dep)
			assert.NoError(t, err)

			assert.Equal(t, "ghcr.io/tacokumo/tacokumo-bot:main-647d918", dep.Spec.Template.Spec.Containers[0].Image)
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
	assert.Equal(t, "ghcr.io/tacokumo/tacokumo-bot:main-647d918", repo.AppConfig.Build.Image)
}

func Test_createOrUpdateObject(t *testing.T) {
	tests := []struct {
		name         string
		existingObj  *appsv1.Deployment
		newObj       *appsv1.Deployment
		expectCreate bool
		expectUpdate bool
	}{
		{
			name:        "create new object when not exists",
			existingObj: nil,
			newObj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-deploy",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To(int32(3)),
				},
			},
			expectCreate: true,
			expectUpdate: false,
		},
		{
			name: "update existing object",
			existingObj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace:       "test-ns",
					Name:            "test-deploy",
					ResourceVersion: "1",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To(int32(1)),
				},
			},
			newObj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "test-ns",
					Name:      "test-deploy",
				},
				Spec: appsv1.DeploymentSpec{
					Replicas: ptr.To(int32(5)),
				},
			},
			expectCreate: false,
			expectUpdate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := runtime.NewScheme()
			err := clientgoscheme.AddToScheme(scheme)
			assert.NoError(t, err)

			clientBuilder := fake.NewClientBuilder().WithScheme(scheme)
			if tt.existingObj != nil {
				clientBuilder = clientBuilder.WithObjects(tt.existingObj)
			}
			k8sClient := clientBuilder.Build()

			// Convert to unstructured
			unstructuredMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(tt.newObj)
			assert.NoError(t, err)

			obj := &unstructured.Unstructured{Object: unstructuredMap}
			obj.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))

			err = createOrUpdateObject(t.Context(), k8sClient, obj)
			assert.NoError(t, err)

			// Verify the result
			result := &appsv1.Deployment{}
			err = k8sClient.Get(t.Context(), types.NamespacedName{
				Namespace: tt.newObj.Namespace,
				Name:      tt.newObj.Name,
			}, result)
			assert.NoError(t, err)

			if tt.expectUpdate && tt.existingObj != nil {
				// ResourceVersion should be preserved or updated
				assert.NotEmpty(t, result.ResourceVersion)
			}
		})
	}
}
