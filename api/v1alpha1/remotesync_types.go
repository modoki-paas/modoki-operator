/*


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

// type Enabled struct {
// 	// +kubebuilder:default:=true
// 	Build bool `json:"build"`

// 	// +kubebuilder:default:=true
// 	Deploy bool `json:"deploy"`
// }

// ApplicationRef is the name of the Application resource
type ApplicationRef struct {
	Name string `json:"name"`
}

// GitHub is the source from GitHub
type GitHub struct {
	// Owner is the repository's owner
	Owner string `json:"owner"`
	// Repository is the repository's name
	Repository string `json:"repo"`

	// SHA is the sha of a commit(optional)
	// +kubebuilder:validation:Optional
	SHA string `json:"sha,omitempty"`

	// Branch is the branch name(optional)
	// +kubebuilder:validation:Optional
	Branch string `json:"branch,omitempty"`

	// PullRequest is the pull request id(optional)
	// +kubebuilder:validation:Optional
	PullRequest *int `json:"pullRequest,omitempty"`

	// SecretName is the name of the Secret resource saving a GitHub token
	SecretName string `json:"secretName"`
}

type Base struct {
	GitHub GitHub `json:"github"`

	// SubPath is the target directory in your repository
	// +kubebuilder:validation:Default="/"
	SubPath string `json:"subPath"`
}

type Image struct {
	Name string `json:"name"`

	// SecretName is the secret to pull from / push to the image registry
	// +kubebuilder:validation:Optional
	SecretName string `json:"secretName"`
}

// RemoteSyncSpec defines the desired state of RemoteSync
type RemoteSyncSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Enabled        Enabled        `json:"enabled"`
	ApplicationRef ApplicationRef `json:"applicationRef"`
	Base           Base           `json:"base"`
	Image          Image          `json:"image"`
}

// RemoteSyncStatus defines the observed state of RemoteSync
type RemoteSyncStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Message is the detailed status or reason for the currnt status
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RemoteSync is the Schema for the remotesyncs API
type RemoteSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RemoteSyncSpec   `json:"spec,omitempty"`
	Status RemoteSyncStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RemoteSyncList contains a list of RemoteSync
type RemoteSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RemoteSync `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RemoteSync{}, &RemoteSyncList{})
}
