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
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// ConfigSpec defines the desired state of Config
type ConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Flags              *Flags    `json:"flags,omitempty"`
	Namespace          Namespace `json:"namespace,omitempty"`
	ServiceAccountName string    `json:"serviceAccountName"`
}

// Namespace represents struct for release namespace data
type Namespace struct {
	Allowed []string `json:"allowed,omitempty"`
	Install bool     `json:"install,omitempty"`
}

type Sync struct {
	Enabled  bool `json:"enabled,omitempty"`
	Interval int  `json:"interval,omitempty"`
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

// ConfigStatus defines the observed state of Config
type ConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
// +kubebuilder:storageversion

// Config is the Schema for the configs API
type Config struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ConfigSpec   `json:"spec,omitempty"`
	Status ConfigStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ConfigList contains a list of Config
type ConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Config `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Config{}, &ConfigList{})
}
