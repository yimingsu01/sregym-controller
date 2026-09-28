/*
Copyright 2026.

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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ResultsStorageSpec struct {
	// +kubebuilder:default="10Gi"
	Size string `json:"size,omitempty"`

	// +optional
	StorageClassName *string `json:"storageClassName,omitempty"`
}

// SREGymExperimentSpec defines the desired state of SREGymExperiment
type SREGymExperimentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// foo is an example field of SREGymExperiment. Edit sregymexperiment_types.go to remove/update
	// +optional
	// Foo *string `json:"foo,omitempty"`

	// +kubebuilder:validation:MinLength=0
	// +required
	ExperimentName string `json:"experimentName"`

	// +kubebuilder:validation:MinLength=0
	// +required
	// Sregym commit to run the experiment with
	ExperimentCommit string `json:"experimentCommit"`

	// +kubebuilder:validation:MinLength=0
	// +required
	Agent string `json:"agent"`

	// +kubebuilder:validation:MinLength=0
	// +required
	Model string `json:"model"`

	// +optional
	// sregym suite in the experiment
	ExperimentSuite string `json:"experimentSuite,omitempty"`

	// +optional
	// +kubebuilder:default="anthropic/claude-sonnet-5"
	// sregym judge model in the experiment
	JudgeModel string `json:"judgeModel,omitempty"`

	// +required
	// +kubebuilder:validation:MinItems=1
	Problems []string `json:"problems"`

	// +optional
	CredentialsSecretRef *corev1.LocalObjectReference `json:"credentialsSecretRef,omitempty"`

	// +required
	ResultsStorage ResultsStorageSpec `json:"resultsStorage"`
}

// SREGymExperimentStatus defines the observed state of SREGymExperiment.
type SREGymExperimentStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties
	// +optional
	ExperimentStartTime *metav1.Time `json:"experimentStartTime,omitempty"`

	// conditions represent the current state of the SREGymExperiment resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SREGymExperiment is the Schema for the sregymexperiments API
type SREGymExperiment struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of SREGymExperiment
	// +required
	Spec SREGymExperimentSpec `json:"spec"`

	// status defines the observed state of SREGymExperiment
	// +optional
	Status SREGymExperimentStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// SREGymExperimentList contains a list of SREGymExperiment
type SREGymExperimentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []SREGymExperiment `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &SREGymExperiment{}, &SREGymExperimentList{})
		return nil
	})
}
