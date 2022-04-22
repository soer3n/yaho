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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// RepositorySpec defines the desired state of Repo
type RepositorySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Name       string  `json:"name"`
	URL        string  `json:"url"`
	Charts     []Entry `json:"charts,omitempty"`
	Sync       Sync    `json:"sync,omitempty"`
	AuthSecret string  `json:"authSecret,omitempty"`
}

// RepositoryStatus defines the observed state of Repo
type RepositoryStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Synced     bool               `json:"synced,omitempty"`
	Charts     Charts             `json:"charts,omitempty"`
	Conditions []metav1.Condition `json:"conditions"`
}

type Charts struct {
	Synced bool `json:"synced,omitempty"`
	Count  int  `json:"count,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Group",type="string",JSONPath=`.metadata.labels['repoGroup']`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Repository is the Schema for the repos API
type Repository struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RepositorySpec   `json:"spec,omitempty"`
	Status RepositoryStatus `json:"status,omitempty"`
}

type Entry struct {
	Name     string   `json:"name,omitempty"`
	Versions []string `json:"versions,omitempty"`
}

// +kubebuilder:object:root=true

// RepositoryList contains a list of Repo
type RepositoryList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Repository `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Repository{}, &RepositoryList{})
}
