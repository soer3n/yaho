/*
Copyright 2021.

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
	"helm.sh/helm/v3/pkg/chart"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ChartSpec defines the desired state of Chart
type ChartSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name string `json:"name,omitempty"`
	// The URL to a relevant project page, git repo, or contact person
	Home string `json:"home,omitempty"`
	// Source is the URL to the source code of this chart
	Sources []string `json:"sources,omitempty"`
	// A SemVer 2 conformant version string of the chart
	Versions []ChartVersion `json:"versions,omitempty"`
	// A one-sentence description of the chart
	Description string `json:"description,omitempty"`
	// A list of string keywords
	Keywords []string `json:"keywords,omitempty"`
	// A list of name and URL/email address combinations for the maintainer(s)
	Maintainers []*chart.Maintainer `json:"maintainers,omitempty"`
	// The URL to an icon file.
	Icon string `json:"icon,omitempty"`
	// The API Version of this chart.
	APIVersion string `json:"apiVersion,omitempty"`
	// The condition to check to enable chart
	Condition string `json:"condition,omitempty"`
	// The tags to check to enable chart
	Tags string `json:"tags,omitempty"`
	// The version of the application enclosed inside of this chart.
	AppVersion string `json:"appVersion,omitempty"`
	// Whether or not this chart is deprecated
	Deprecated bool `json:"deprecated,omitempty"`
	// Annotations are additional mappings uninterpreted by Helm,
	// made available for inspection by other applications.
	Annotations map[string]string `json:"annotations,omitempty"`
	// KubeVersion is a SemVer constraint specifying the version of Kubernetes required.
	KubeVersion string `json:"kubeVersion,omitempty"`
	// Specifies the chart type: application or library
	Type string `json:"type,omitempty"`
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
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Group",type="string",JSONPath=`.metadata.labels['repoGroup']`
// +kubebuilder:printcolumn:name="Repo",type="string",JSONPath=`.metadata.labels['repo']`
// +kubebuilder:printcolumn:name="Created_at",type="string",JSONPath=`.metadata.creationTimestamp`

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
