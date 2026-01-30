package application

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	tacokumogithubiov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/repoconnector"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func testdataPath(subpath string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "repoconnector", "testdata", subpath)
}

func newTestScheme(t *testing.T) *k8sruntime.Scheme {
	t.Helper()
	scheme := k8sruntime.NewScheme()
	err := tacokumogithubiov1alpha1.AddToScheme(scheme)
	require.NoError(t, err)
	return scheme
}

func newTestManager(t *testing.T, k8sClient client.Client, connector repoconnector.GitRepositoryConnector) *Manager {
	t.Helper()
	m := NewManager(logr.Discard(), k8sClient)
	if connector != nil {
		m.WithConnector(connector)
	}
	return m
}

func TestManager_Reconcile_OnProvisioningState(t *testing.T) {
	tests := []struct {
		name                 string
		testdataDir          string
		appConfigPath        string
		expectedState        string
		expectError          bool
		expectedReleaseCount int
	}{
		{
			name:                 "valid appconfig with stages creates releases and transitions to Waiting",
			testdataDir:          "valid-appconfig",
			appConfigPath:        "appconfig.yaml",
			expectedState:        tacokumogithubiov1alpha1.ApplicationStateWaiting,
			expectError:          false,
			expectedReleaseCount: 2, // staging, production
		},
		{
			name:                 "empty stages uses default stages",
			testdataDir:          "empty-stages",
			appConfigPath:        "appconfig.yaml",
			expectedState:        tacokumogithubiov1alpha1.ApplicationStateWaiting,
			expectError:          false,
			expectedReleaseCount: 1, // production (default)
		},
		{
			name:                 "non-existent appconfig file causes error",
			testdataDir:          "valid-appconfig",
			appConfigPath:        "non-existent.yaml",
			expectedState:        tacokumogithubiov1alpha1.ApplicationStateError,
			expectError:          true,
			expectedReleaseCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme(t)
			app := &tacokumogithubiov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-app",
				},
				Spec: tacokumogithubiov1alpha1.ApplicationSpec{
					ReleaseTemplate: tacokumogithubiov1alpha1.ReleaseSpec{
						AppConfigPath: tt.appConfigPath,
					},
				},
				Status: tacokumogithubiov1alpha1.ApplicationStatus{
					State: tacokumogithubiov1alpha1.ApplicationStateProvisioning,
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(app).
				WithStatusSubresource(app).
				Build()

			connector := repoconnector.NewLocalConnector(testdataPath(tt.testdataDir))
			m := newTestManager(t, k8sClient, connector)

			err := m.Reconcile(t.Context(), app)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedState, app.Status.State)
			assert.Len(t, app.Status.Releases, tt.expectedReleaseCount)
		})
	}
}

func TestManager_Reconcile_OnWaitingState(t *testing.T) {
	tests := []struct {
		name          string
		releaseStates []string
		expectedState string
		expectError   bool
	}{
		{
			name: "all releases deployed transitions to Running",
			releaseStates: []string{
				tacokumogithubiov1alpha1.ReleaseStateDeployed,
				tacokumogithubiov1alpha1.ReleaseStateDeployed,
			},
			expectedState: tacokumogithubiov1alpha1.ApplicationStateRunning,
			expectError:   false,
		},
		{
			name: "some releases still deploying stays in Waiting",
			releaseStates: []string{
				tacokumogithubiov1alpha1.ReleaseStateDeployed,
				tacokumogithubiov1alpha1.ReleaseStateDeploying,
			},
			expectedState: tacokumogithubiov1alpha1.ApplicationStateWaiting,
			expectError:   false,
		},
		{
			name:          "no releases transitions to Running",
			releaseStates: []string{},
			expectedState: tacokumogithubiov1alpha1.ApplicationStateRunning,
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme(t)

			// Create releases with given states
			releases := make([]corev1.ObjectReference, 0, len(tt.releaseStates))
			objects := make([]client.Object, 0, len(tt.releaseStates)+1)

			for i, state := range tt.releaseStates {
				rel := &tacokumogithubiov1alpha1.Release{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "default",
						Name:      fmt.Sprintf("test-app-stage-%d", i),
					},
					Status: tacokumogithubiov1alpha1.ReleaseStatus{
						State: state,
					},
				}
				objects = append(objects, rel)
				releases = append(releases, corev1.ObjectReference{
					Namespace: rel.Namespace,
					Name:      rel.Name,
				})
			}

			app := &tacokumogithubiov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-app",
				},
				Status: tacokumogithubiov1alpha1.ApplicationStatus{
					State:    tacokumogithubiov1alpha1.ApplicationStateWaiting,
					Releases: releases,
				},
			}
			objects = append(objects, app)

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(objects...).
				WithStatusSubresource(app).
				Build()

			m := newTestManager(t, k8sClient, nil)

			err := m.Reconcile(t.Context(), app)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedState, app.Status.State)
		})
	}
}

func TestManager_Reconcile_OnWaitingState_ReleaseNotFound(t *testing.T) {
	scheme := newTestScheme(t)

	app := &tacokumogithubiov1alpha1.Application{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "default",
			Name:      "test-app",
		},
		Status: tacokumogithubiov1alpha1.ApplicationStatus{
			State: tacokumogithubiov1alpha1.ApplicationStateWaiting,
			Releases: []corev1.ObjectReference{
				{
					Namespace: "default",
					Name:      "non-existent-release",
				},
			},
		},
	}

	k8sClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(app).
		WithStatusSubresource(app).
		Build()

	m := newTestManager(t, k8sClient, nil)

	err := m.Reconcile(t.Context(), app)

	assert.Error(t, err)
	assert.Equal(t, tacokumogithubiov1alpha1.ApplicationStateError, app.Status.State)
}

func TestManager_Reconcile_StaysInTerminalState(t *testing.T) {
	tests := []struct {
		name  string
		state string
	}{
		{
			name:  "stays in Running state",
			state: tacokumogithubiov1alpha1.ApplicationStateRunning,
		},
		{
			name:  "stays in Error state",
			state: tacokumogithubiov1alpha1.ApplicationStateError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme(t)

			app := &tacokumogithubiov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-app",
				},
				Status: tacokumogithubiov1alpha1.ApplicationStatus{
					State: tt.state,
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(app).
				WithStatusSubresource(app).
				Build()

			m := newTestManager(t, k8sClient, nil)

			err := m.Reconcile(t.Context(), app)

			assert.NoError(t, err)
			assert.Equal(t, tt.state, app.Status.State)
		})
	}
}

func TestManager_Reconcile_OnDefaultState(t *testing.T) {
	tests := []struct {
		name          string
		initialState  string
		expectedState string
	}{
		{
			name:          "empty state transitions to Provisioning",
			initialState:  "",
			expectedState: tacokumogithubiov1alpha1.ApplicationStateProvisioning,
		},
		{
			name:          "unknown state transitions to Provisioning",
			initialState:  "Unknown",
			expectedState: tacokumogithubiov1alpha1.ApplicationStateProvisioning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme(t)

			app := &tacokumogithubiov1alpha1.Application{
				ObjectMeta: metav1.ObjectMeta{
					Namespace: "default",
					Name:      "test-app",
				},
				Status: tacokumogithubiov1alpha1.ApplicationStatus{
					State: tt.initialState,
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(app).
				WithStatusSubresource(app).
				Build()

			m := newTestManager(t, k8sClient, nil)

			err := m.Reconcile(t.Context(), app)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedState, app.Status.State)
		})
	}
}
