package helmutil_test

import (
	"testing"

	"github.com/tacokumo/portal-controller-kubernetes/pkg/helmutil"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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

			err = helmutil.CreateOrUpdateObject(t.Context(), k8sClient, obj)
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
