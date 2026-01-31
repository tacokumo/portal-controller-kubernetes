package release

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	tacokumogithubiov1alpha1 "github.com/tacokumo/portal-controller-kubernetes/api/v1alpha1"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/appconfig"
	"github.com/tacokumo/portal-controller-kubernetes/pkg/repoconnector"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apispec "github.com/tacokumo/api-spec"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// Helper functions

func testdataPath(subpath string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "testdata", subpath)
}

func repoTestdataPath(subpath string) string {
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

func newTestManager(
	t *testing.T,
	k8sClient client.Client,
	connector repoconnector.GitRepositoryConnector,
	workdir string,
) *Manager {
	t.Helper()
	m := NewManager(logr.Discard(), k8sClient, workdir)
	if connector != nil {
		m.WithConnector(connector)
	}
	return m
}

// Tests for constructReleaseValues

func TestManager_constructReleaseValues(t *testing.T) {
	tests := []struct {
		name         string
		releaseName  string
		image        string
		expectError  bool
		validateFunc func(*testing.T, map[string]interface{})
	}{
		{
			name:        "populates values correctly with image",
			releaseName: "test-release",
			image:       "myregistry.example.com/app:v1.0.0",
			expectError: false,
			validateFunc: func(t *testing.T, values map[string]interface{}) {
				require.NotNil(t, values)
				main, ok := values["main"].(map[string]interface{})
				require.True(t, ok, "main should be a map")

				assert.Equal(t, "test-release", main["applicationName"])
				assert.Equal(t, "myregistry.example.com/app:v1.0.0", main["image"])
				assert.Equal(t, 1, main["replicaCount"])
			},
		},
		{
			name:        "handles empty image string",
			releaseName: "empty-image-release",
			image:       "",
			expectError: false,
			validateFunc: func(t *testing.T, values map[string]interface{}) {
				main, ok := values["main"].(map[string]interface{})
				require.True(t, ok)
				assert.Equal(t, "", main["image"])
			},
		},
		{
			name:        "sets replica count to 1",
			releaseName: "replica-test",
			image:       "test-image:latest",
			expectError: false,
			validateFunc: func(t *testing.T, values map[string]interface{}) {
				main := values["main"].(map[string]interface{})
				assert.Equal(t, 1, main["replicaCount"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme(t)
			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			m := newTestManager(t, k8sClient, nil, "/tmp/test")

			rel := &tacokumogithubiov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.releaseName,
					Namespace: "default",
				},
			}

			repo := &appconfig.Repository{
				AppConfig: apispec.AppConfig{
					Build: apispec.BuildConfig{
						Image: tt.image,
					},
				},
			}

			values, err := m.constructReleaseValues(rel, repo)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				tt.validateFunc(t, values)
			}
		})
	}
}

// Tests for handleError

func TestManager_handleError(t *testing.T) {
	tests := []struct {
		name               string
		initialState       string
		originalError      error
		expectStatusUpdate bool
		expectedFinalState string
	}{
		{
			name:               "sets state to ReleaseStateFailed",
			initialState:       tacokumogithubiov1alpha1.ReleaseStateDeploying,
			originalError:      fmt.Errorf("deployment failed"),
			expectStatusUpdate: true,
			expectedFinalState: tacokumogithubiov1alpha1.ReleaseStateFailed,
		},
		{
			name:               "preserves original error",
			initialState:       tacokumogithubiov1alpha1.ReleaseStateDeploying,
			originalError:      fmt.Errorf("test error"),
			expectStatusUpdate: true,
			expectedFinalState: tacokumogithubiov1alpha1.ReleaseStateFailed,
		},
		{
			name:               "updates status even when already in failed state",
			initialState:       tacokumogithubiov1alpha1.ReleaseStateFailed,
			originalError:      fmt.Errorf("another error"),
			expectStatusUpdate: true,
			expectedFinalState: tacokumogithubiov1alpha1.ReleaseStateFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme(t)

			rel := &tacokumogithubiov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-release",
					Namespace: "default",
				},
				Status: tacokumogithubiov1alpha1.ReleaseStatus{
					State: tt.initialState,
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(rel).
				WithStatusSubresource(rel).
				Build()

			m := newTestManager(t, k8sClient, nil, "/tmp/test")

			err := m.handleError(context.Background(), rel, tt.originalError)

			// Should return the original error
			assert.Equal(t, tt.originalError, err)

			// Should update state to Failed
			assert.Equal(t, tt.expectedFinalState, rel.Status.State)

			// Verify status was updated in the client
			updatedRel := &tacokumogithubiov1alpha1.Release{}
			err = k8sClient.Get(context.Background(), client.ObjectKey{
				Namespace: rel.Namespace,
				Name:      rel.Name,
			}, updatedRel)
			require.NoError(t, err)
			assert.Equal(t, tt.expectedFinalState, updatedRel.Status.State)
		})
	}
}

// Tests for Reconcile state machine

func TestManager_Reconcile_OnDefaultState(t *testing.T) {
	tests := []struct {
		name          string
		initialState  string
		expectedState string
	}{
		{
			name:          "empty state transitions to Deploying",
			initialState:  "",
			expectedState: tacokumogithubiov1alpha1.ReleaseStateDeploying,
		},
		{
			name:          "unknown state transitions to Deploying",
			initialState:  "Unknown",
			expectedState: tacokumogithubiov1alpha1.ReleaseStateDeploying,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme(t)

			rel := &tacokumogithubiov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-release",
					Namespace: "default",
				},
				Status: tacokumogithubiov1alpha1.ReleaseStatus{
					State: tt.initialState,
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(rel).
				WithStatusSubresource(rel).
				Build()

			m := newTestManager(t, k8sClient, nil, "/tmp/test")

			err := m.Reconcile(context.Background(), rel)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedState, rel.Status.State)
		})
	}
}

// Tests for reconcileOnDeployingState

func TestManager_reconcileOnDeployingState(t *testing.T) {
	// Use testdata directory as workdir
	workdir := testdataPath("")

	tests := []struct {
		name             string
		testdataDir      string
		appConfigPath    string
		releaseName      string
		releaseNamespace string
		appConfigBranch  string
		commit           *string
		setupChart       bool
		expectError      bool
	}{
		{
			name:             "successfully deploys with valid appconfig",
			testdataDir:      "release-test-data",
			appConfigPath:    "appconfig.yaml",
			releaseName:      "test-app-production",
			releaseNamespace: "production",
			appConfigBranch:  "main",
			setupChart:       true,
			expectError:      false,
		},
		{
			name:             "uses commit when appConfigBranch is empty",
			testdataDir:      "release-test-data",
			appConfigPath:    "appconfig.yaml",
			releaseName:      "commit-test",
			releaseNamespace: "default",
			appConfigBranch:  "",
			commit:           stringPtr("abc123"),
			setupChart:       true,
			expectError:      false,
		},
		{
			name:             "fails when appconfig not found",
			testdataDir:      "valid-appconfig",
			appConfigPath:    "nonexistent.yaml",
			releaseName:      "fail-test",
			releaseNamespace: "default",
			appConfigBranch:  "main",
			setupChart:       true,
			expectError:      true,
		},
		{
			name:             "passes correct namespace to RenderChart",
			testdataDir:      "release-test-data",
			appConfigPath:    "appconfig.yaml",
			releaseName:      "namespace-test",
			releaseNamespace: "custom-namespace",
			appConfigBranch:  "main",
			setupChart:       true,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme := newTestScheme(t)

			rel := &tacokumogithubiov1alpha1.Release{
				ObjectMeta: metav1.ObjectMeta{
					Name:      tt.releaseName,
					Namespace: tt.releaseNamespace,
				},
				Spec: tacokumogithubiov1alpha1.ReleaseSpec{
					Repo: tacokumogithubiov1alpha1.RepositoryRef{
						URL: "https://github.com/test/repo.git",
					},
					AppConfigPath:   tt.appConfigPath,
					AppConfigBranch: tt.appConfigBranch,
					Commit:          tt.commit,
				},
				Status: tacokumogithubiov1alpha1.ReleaseStatus{
					State: tacokumogithubiov1alpha1.ReleaseStateDeploying,
				},
			}

			k8sClient := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(rel).
				WithStatusSubresource(rel).
				Build()

			connector := repoconnector.NewLocalConnector(repoTestdataPath(tt.testdataDir))

			m := newTestManager(t, k8sClient, connector, workdir)

			err := m.reconcileOnDeployingState(context.Background(), rel)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func stringPtr(s string) *string {
	return &s
}
