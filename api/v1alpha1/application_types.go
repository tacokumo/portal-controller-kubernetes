/*
Copyright 2025 drumato.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	Repo          RepositoryRef `json:"repo"`
	AppConfigPath string        `json:"appConfigPath,omitempty"`
}

type RepositoryRef struct {
	URL string `json:"url"`
	Ref string `json:"ref"`
}

// ApplicationStatus defines the observed state of Application.
type ApplicationStatus struct {
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	Deployments []NamespacedName `json:"deployments,omitempty"`
	State       string           `json:"state,omitempty"`
}

const (
	// ApplicationStateProvisioning は差分検知などによって遷移し､
	// 現在の定義によってアプリケーション状態の収束を再度キックすることを示します
	ApplicationStateProvisioning = "Provisioning"
	// ApplicationStateWaiting はアプリケーションの稼働準備中であることを示します
	// このStateではすべてのアプリケーションがReadyになっていること、
	// Appconfigのhealthcheckが成功することを期待します
	ApplicationStateWaiting = "Waiting"
	// ApplicationStateRunning はアプリケーションが正常に稼働していることを示します
	ApplicationStateRunning = "Running"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

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
