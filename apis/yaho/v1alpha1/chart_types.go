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

// ChartSpec defines the desired state of Chart
type ChartSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name       string `json:"name,omitempty"`
	Repository string `json:"repository"`
	// A SemVer 2 conformant version string of the chart
	Versions []string `json:"versions,omitempty"`
	// The tags to check to enable chart
	CreateDeps bool `json:"createDeps,omitempty"`
}

// ChartDep represents data for parsing a chart dependency
type ChartDep struct {
	Name      string `json:"name,omitempty"`
	Version   string `json:"version,omitempty"`
	Repo      string `json:"repo,omitempty"`
	Condition string `json:"condition,omitempty"`
}

// ChartVersion repesents data for parsing a chart
type ChartVersion struct {
	Name         string      `json:"name"`
	Templates    string      `json:"templateRef"`
	CRDs         string      `json:"crdRef,omitempty"`
	Dependencies []*ChartDep `json:"deps,omitempty"`
	URL          string      `json:"url,omitempty"`
}

// ChartStatus defines the observed state of Chart
type ChartStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Dependencies *string            `json:"dependencies,omitempty"`
	Versions     *string            `json:"versions,omitempty"`
	Conditions   []metav1.Condition `json:"conditions"`
	Deprecated   *bool              `json:"deprecated,omitempty"`
	Type         *string            `json:"type,omitempty"`
	Tags         *string            `json:"tags,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Group",type="string",JSONPath=`.metadata.labels['repoGroup']`
// +kubebuilder:printcolumn:name="Repo",type="string",JSONPath=`.spec.repository`
// +kubebuilder:printcolumn:name="Versions",type="string",JSONPath=`.status.versions`
// +kubebuilder:printcolumn:name="Deps",type="string",JSONPath=`.status.dependencies`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Chart is the Schema for the charts API
type Chart struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChartSpec   `json:"spec,omitempty"`
	Status ChartStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ChartList contains a list of Chart
type ChartList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Chart `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Chart{}, &ChartList{})
}
