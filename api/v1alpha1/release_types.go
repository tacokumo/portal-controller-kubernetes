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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReleaseSpec defines the desired state of Release
type ReleaseSpec struct {
	// Repo はappconfigが格納されているGitリポジトリを示します
	Repo RepositoryRef `json:"repo"`
	// AppConfigPath はappconfig.yamlが格納されているパスを示します
	AppConfigPath string `json:"appConfigPath,omitempty"`
	// AppConfigBranch はappconfigが格納されているGitブランチを示します
	// 何も指定されない場合はmainが使用されます
	AppConfigBranch string `json:"appConfigBranch,omitempty"`
	// Commit はReleaseに使用するGitコミットハッシュを示します
	Commit *string `json:"commit,omitempty"`
	// APIでアプリケーションに対し環境変数をセットされたときに、
	// それが格納されたSecretが存在する仮定する
	EnvSecretName *string `json:"envSecretName,omitempty"`
}

// ReleaseStatus defines the observed state of Release.
type ReleaseStatus struct {
	State string `json:"state,omitempty"`
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

const (
	// ReleaseStateDeploying は差分検知などによって遷移し､
	// Releaseのリソースをデプロイ中であることを示します
	ReleaseStateDeploying = "Deploying"

	// ReleaseStateDeployed はReleaseのリソースが正常にデプロイされたことを示します
	ReleaseStateDeployed = "Deployed"

	// ReleaseStateFailed はReleaseのリソースのデプロイが失敗したことを示します
	ReleaseStateFailed = "Failed"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Release is the Schema for the releases API
type Release struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Release
	// +required
	Spec ReleaseSpec `json:"spec"`

	// status defines the observed state of Release
	// +optional
	Status ReleaseStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ReleaseList contains a list of Release
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Release `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
