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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ReleaseSpec defines the desired state of Release
type ReleaseSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Name           string         `json:"name"`
	Namespace      Namespace      `json:"namespace,omitempty"`
	Repo           string         `json:"repo"`
	Chart          string         `json:"chart"`
	Version        string         `json:"version,omitempty"`
	ValuesTemplate *ValueTemplate `json:"releaseSpec,omitempty"`
	Flags          *Flags         `json:"flags,omitempty"`
}

// ValueTemplate represents data for install process of a release
type ValueTemplate struct {
	ValueRefs          []string                    `json:"valueRefs,omitempty"`
	DependenciesConfig map[string]DependencyConfig `json:"deps,omitempty"`
}

// Flags represents data for parsing flags for creating release resources
type Flags struct {
	Atomic                   bool          `json:"atomic,omitempty"`
	SkipCRDs                 bool          `json:"skipCRDs,omitempty"`
	SubNotes                 bool          `json:"subNotes,omitempty"`
	DisableOpenAPIValidation bool          `json:"disableOpenAPIValidation,omitempty"`
	DryRun                   bool          `json:"dryRun,omitempty"`
	DisableHooks             bool          `json:"disableHooks,omitempty"`
	Wait                     bool          `json:"wait,omitempty"`
	Timeout                  time.Duration `json:"timeout,omitempty"`
	Force                    bool          `json:"force,omitempty"`
	Description              string        `json:"description,omitempty"`
	Recreate                 bool          `json:"recreate,omitempty"`
	CleanupOnFail            bool          `json:"cleanupOnFail,omitempty"`
}

// DependencyConfig represents data for a chart dependency in a release
type DependencyConfig struct {
	Enabled bool   `json:"enabled,omitempty"`
	Values  string `json:"values,omitempty"`
}

// Namespace represents struct for release namespace data
type Namespace struct {
	Name    string `json:"name,omitempty"`
	Install bool   `json:"install,omitempty"`
}

// ReleaseStatus defines the observed state of Release
type ReleaseStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	Synced     string             `json:"synced,omitempty"`
	Conditions []metav1.Condition `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Group",type="string",JSONPath=`.metadata.labels['repoGroup']`
// +kubebuilder:printcolumn:name="Repo",type="string",JSONPath=`.spec.repo`
// +kubebuilder:printcolumn:name="Chart",type="string",JSONPath=`.spec.chart`
// +kubebuilder:printcolumn:name="Created_at",type="string",JSONPath=`.metadata.creationTimestamp`

// Release is the Schema for the releases API
type Release struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReleaseSpec   `json:"spec,omitempty"`
	Status ReleaseStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReleaseList contains a list of Release
type ReleaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Release `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Release{}, &ReleaseList{})
}
