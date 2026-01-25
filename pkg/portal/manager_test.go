package portal

import (
	"testing"

	tacokumoiov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestManager_reconcileOnProvisioningState(t *testing.T) {
	tests := []struct {
		name          string
		p             *tacokumoiov1alpha1.Portal
		expectedState string
	}{
		{
			name: "basic",
			p: &tacokumoiov1alpha1.Portal{
				ObjectMeta: metav1.ObjectMeta{
					Name: "basic",
				},
			},
			expectedState: tacokumoiov1alpha1.PortalStateWaiting,
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

			err = manager.reconcileOnProvisioningState(t.Context(), tt.p)
			assert.NoError(t, err)
			assert.Equal(t, tt.expectedState, tt.p.Status.State)

			ns := corev1.Namespace{}
			err = k8sClient.Get(t.Context(), types.NamespacedName{Name: tt.p.Name}, &ns)
			assert.NoError(t, err)

			dep := appsv1.Deployment{}
			err = k8sClient.Get(t.Context(), types.NamespacedName{Namespace: tt.p.Name, Name: tt.p.Name + "-portal-ui"}, &dep)
			assert.NoError(t, err)
		})
	}
}
