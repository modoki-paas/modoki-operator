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

type MetadataTemplate struct {
	// Map of string keys and values that can be used to organize and categorize
	// (scope and select) objects. May match selectors of replication controllers
	// and services.
	// More info: http://kubernetes.io/docs/user-guide/labels
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations is an unstructured key value map stored with a resource that may be
	// set by external tools to store and retrieve arbitrary metadata. They are not
	// queryable and should be preserved when modifying objects.
	// More info: http://kubernetes.io/docs/user-guide/annotations
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ApplicationSpecTemplate defines the desired state of Application
type ApplicationSpecTemplate struct {
	// Command is an entrypoint array
	// +kubebuilder:validation:Optional
	Command []string `json:"command,omitempty"`

	// Args is the arguments to the entrypoint
	// +kubebuilder:validation:Optional
	Args []string `json:"args,omitempty"`

	// Attributes is parameters for the generator
	// +kubebuilder:validation:Optional
	Attributes map[string]string `json:"attributes,omitempty"`
}

type ApplicationTemplate struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +kubebuilder:validation:Optional
	MetadataTemplate `json:"metadata,omitempty"`

	Spec ApplicationSpecTemplate `json:"spec"`
}

// GitHubPipeline is the source from GitHub
type GitHubPipeline struct {
	// Owner is the repository's owner
	Owner string `json:"owner"`
	// Repository is the repository's name
	Repository string `json:"repo"`

	// SecretName is the name of the Secret resource saving a GitHub token
	SecretName string `json:"secretName"`
}

type PipelineBase struct {
	GitHub GitHubPipeline `json:"github"`

	// SubPath is the target directory in your repository
	// +kubebuilder:validation:Optional
	SubPath string `json:"subPath"`
}

// AppPipelineSpec defines the desired state of AppPipeline
type AppPipelineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	DomainBase          string              `json:"domainBase"`
	ApplicationTemplate ApplicationTemplate `json:"applicationTemplate"`
	Base                PipelineBase        `json:"base"`
	Image               Image               `json:"image"`
}

// AppPipelineStatus defines the observed state of AppPipeline
type AppPipelineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Message is the detailed status or reason for the currnt status
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// AppPipeline is the Schema for the apppipelines API
type AppPipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AppPipelineSpec   `json:"spec,omitempty"`
	Status AppPipelineStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// AppPipelineList contains a list of AppPipeline
type AppPipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AppPipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&AppPipeline{}, &AppPipelineList{})
}
