/*
Copyright 2021.

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program.  If not, see <http://www.gnu.org/licenses/>.
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// +kubebuilder:pruning:PreserveUnknownFields

// ValuesSpec defines the desired state of Values
type ValuesSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Values   map[string]string `json:"values,omitempty"`
	Refs     map[string]string `json:"refs,omitempty"`
	Selector string            `json:"selector,omitempty"`

	// +kubebuilder:validation:any
	// +kubebuilder:pruning:PreserveUnknownFields

	ValuesMap *runtime.RawExtension `json:"json,omitempty"`
}

// ValuesStatus defines the observed state of Values
type ValuesStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Values is the Schema for the values API
type Values struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ValuesSpec   `json:"spec,omitempty"`
	Status ValuesStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ValuesList contains a list of Values
type ValuesList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Values `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Values{}, &ValuesList{})
}
