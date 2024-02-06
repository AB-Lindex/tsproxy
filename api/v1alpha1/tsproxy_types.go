/*
Copyright 2024.

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

type TSProxyService struct {
	//+required
	// Name of the service to proxy
	Name string `json:"name"`

	//+required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:ExclusiveMinimum=false
	// +kubebuilder:validation:ExclusiveMaximum=false
	// ServicePort contains the port on the service to proxy
	ServicePort int32 `json:"port"`

	//+required
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=65535
	// +kubebuilder:validation:ExclusiveMinimum=false
	// +kubebuilder:validation:ExclusiveMaximum=false
	// ExposeAs contains the port to expose the proxy on the host network
	ExposeAs int32 `json:"exposeAs"`
}

// TSProxySpec defines the desired state of TSProxy
type TSProxySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Services []TSProxyService `json:"services,omitempty"`
}

// TSProxyStatus defines the observed state of TSProxy
type TSProxyStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TSProxy is the Schema for the tsproxies API
type TSProxy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TSProxySpec   `json:"spec,omitempty"`
	Status TSProxyStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TSProxyList contains a list of TSProxy
type TSProxyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TSProxy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TSProxy{}, &TSProxyList{})
}
