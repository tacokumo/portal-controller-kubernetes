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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// ReleaseTemplate は 各Stageに対応するReleaseのテンプレートを示します
	ReleaseTemplate ReleaseSpec `json:"releaseTemplate,omitempty"`
}

// RepositoryRef defines a reference to a Git repository.
type RepositoryRef struct {
	// URL はGitリポジトリのURLを示します
	URL string `json:"url"`
}

// ApplicationStatus defines the observed state of Application.
type ApplicationStatus struct {
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	Releases []corev1.ObjectReference `json:"releases,omitempty"`
	State    string                   `json:"state,omitempty"`
}

const (
	// ApplicationStateProvisioning は差分検知などによって遷移し､
	// appconfigなどを読み込んでReleaseを作成している途中であることを示します
	ApplicationStateProvisioning = "Provisioning"
	// ApplicationStateWaiting はそれぞれのRelease状態が収束するのを待っていることを示します
	ApplicationStateWaiting = "Waiting"
	// ApplicationStateRunning はすべてのReleaseが正常に稼働していることを示します
	ApplicationStateRunning = "Running"
	// ApplicationStateError は何らかのエラーが発生していることを示します
	// Conditions に詳細が記載されます
	ApplicationStateError = "Error"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="STATE",type=string,JSONPath=`.status.state`,description="Current state of the Application"
// +kubebuilder:printcolumn:name="RELEASES",type=string,JSONPath=`.status.releases[*].name`,description="Release names",priority=1
// +kubebuilder:printcolumn:name="READY",type=string,JSONPath=`.status.conditions[?(@.type=="Ready")].status`,description="Ready condition status"
// +kubebuilder:printcolumn:name="AGE",type=date,JSONPath=`.metadata.creationTimestamp`

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of Application
	// +required
	Spec ApplicationSpec `json:"spec"`

	// status defines the observed state of Application
	// +optional
	Status ApplicationStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// ApplicationList contains a list of Application
type ApplicationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Application `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Application{}, &ApplicationList{})
}
