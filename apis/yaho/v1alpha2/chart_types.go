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

// ChartStatus defines the observed state of Chart
type ChartStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	ChartVersions map[string]ChartVersion `json:"chartVersions,omitempty"`
	LinkedCharts  []string                `json:"linkedCharts,omitempty"`
	// TODO: implement conditions in a new way:
	// enum: indexLoaded, configmapCreate, remoteSync, prepareChart, dependenciesSync
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	Deprecated *bool              `json:"deprecated,omitempty"`
}

type ChartVersion struct {
	Loaded    bool `json:"loaded,omitempty"`
	Specified bool `json:"specified,omitempty"`
	// LatestUpdate time.Time `json:"latestUpdate,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:storageversion
// +kubebuilder:printcolumn:name="Repository",type="string",JSONPath=`.spec.repository`
// +kubebuilder:printcolumn:name="Versions",type="string",JSONPath=".status.conditions[?(@.type==\"indexLoaded\")].status"
// +kubebuilder:printcolumn:name="Dependencies",type="string",JSONPath=".status.conditions[?(@.type==\"dependenciesSync\")].status"
// +kubebuilder:printcolumn:name="Deprecated",type="string",JSONPath=`.status.deprecated`
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Chart is the Schema for the charts API
type Chart struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ChartSpec   `json:"spec,omitempty"`
	Status ChartStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ChartList contains a list of Chart
type ChartList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Chart `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Chart{}, &ChartList{})
}
