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

// ApplicationSpec defines the desired state of Application
type ApplicationSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Domains are requested domains for the ingress of the application
	Domains []string `json:"domains"`

	// Image is the url for Docker registry
	// +kubebuilder:validation:Optional
	Image string `json:"image,omitempty"`

	// Command is an entrypoint array
	// +kubebuilder:validation:Optional
	Command []string `json:"command,omitempty"`

	// Args is the arguments to the entrypoint
	// +kubebuilder:validation:Optional
	Args []string `json:"args,omitempty"`

	// Attributes is parameters for the generator
	// +kubebuilder:validation:Optional
	Attributes map[string]string `json:"attributes,omitempty"`

	// ServiceAccount is the name of the ServiceAccount to use to run this Application
	// +kubebuilder:validation:Optional
	ServiceAccount string `json:"serviceAccount,omitempty"`

	// ImagePullSecret is the name of the ImagePullSecret to pull your image
	// +kubebuilder:validation:Optional
	ImagePullSecret string `json:"imagePullSecret,omitempty"`
}

// ApplicationStatus defines the observed state of Application
type ApplicationStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Domains are assigned domains for the application
	Domains []string `json:"domains"`

	// Status is the current status of the application
	Status ApplicationStatusType `json:"status"`

	// Message is the detailed status or reason for the currnt status
	// +kubebuilder:validation:Optional
	Message string `json:"message,omitempty"`

	// Resources are the generated resources by modoki
	Resources []ApplicationResource `json:"resources"`
}

// ApplicationResource is a resource in Kubernetes
type ApplicationResource struct {
	metav1.TypeMeta `json:",inline"`
	Name            string `json:"name"`
	Namespace       string `json:"namespace,omitempty"`
}

// ApplicationStatusType is the Status enum for Application
type ApplicationStatusType string

const (
	// ApplicationDeployed means all updated specs are applied
	ApplicationDeployed ApplicationStatusType = "deployed"

	// ApplicationProgressing means specs are being updated
	ApplicationProgressing ApplicationStatusType = "progressing"

	// ApplicationDeploymentFailed means deployment failed
	ApplicationDeploymentFailed ApplicationStatusType = "failed"

	// ApplicationError means some errors occurred in the application
	ApplicationError ApplicationStatusType = "error"
)

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Application is the Schema for the applications API
type Application struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ApplicationSpec   `json:"spec,omitempty"`
	Status ApplicationStatus `json:"status,omitempty"`
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
