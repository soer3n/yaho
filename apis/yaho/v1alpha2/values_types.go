/*
Copyright 2023.

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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ValuesSpec defines the desired state of Values
type ValuesSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Values   map[string]string `json:"values,omitempty"`
	Refs     map[string]string `json:"refs,omitempty"`
	Selector string            `json:"selector,omitempty"`

	// +kubebuilder:validation:any
	// +kubebuilder:pruning:PreserveUnknownFields

	ValuesMap *runtime.RawExtension `json:"yaml,omitempty"`
}

// ValuesStatus defines the observed state of Values
type ValuesStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:storageversion

// Values is the Schema for the values API
type Values struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ValuesSpec   `json:"spec,omitempty"`
	Status ValuesStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ValuesList contains a list of Values
type ValuesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Values `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Values{}, &ValuesList{})
}