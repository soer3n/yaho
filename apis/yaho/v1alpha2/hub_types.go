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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// HubSpec defines the desired state of Hub
type HubSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Clusters []HubCluster `json:"clusters,omitempty"`
	Interval string       `json:"interval,omitempty"`
	Defaults HubDefaults  `json:"defaults,omitempty"`
}

type HubCluster struct {
	Name     string           `json:"name,omitempty"`
	Agent    *HubClusterAgent `json:"agent,omitempty"`
	Secret   Secret           `json:"secret,omitempty"`
	Defaults HubDefaults      `json:"defaults,omitempty"`
}

type HubClusterAgent struct {
	Deploy    *bool  `json:"deploy,omitempty"`
	Name      string `json:"name,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type Secret struct {
	Name      string `json:"name,omitempty"`
	Key       string `json:"key,omitempty"`
	Namespace string `json:"namespace,omitempty"`
}

type HubDefaults struct{}

// HubStatus defines the observed state of Hub
type HubStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Backends map[string]HubBackend `json:"backends,omitempty"`
}

type HubBackend struct {
	Address string `json:"address,omitempty"`
	InSync  bool   `json:"synced,omitempty"`
}

//+kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
//+kubebuilder:subresource:status
// +kubebuilder:storageversion

// Hub is the Schema for the hubs API
type Hub struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   HubSpec   `json:"spec,omitempty"`
	Status HubStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// HubList contains a list of Hub
type HubList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Hub `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Hub{}, &HubList{})
}
