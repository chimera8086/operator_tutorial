/*
Copyright 2025.

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// SecSynchSpec defines the desired state of SecSynch.
type SecSynchSpec struct {
	SourceNamespace       string   `json:"sourceNamespace"`
	DestinationNamespaces []string `json:"destinationNamespaces"`
	SecretName            string   `json:"secretName"`
}

type SecSynchStatus struct {
	LastSyncTime metav1.Time `json:"lastSyncTime"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// SecSynch is the Schema for the secsynches API.
type SecSynch struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SecSynchSpec   `json:"spec,omitempty"`
	Status SecSynchStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SecSynchList contains a list of SecSynch.
type SecSynchList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecSynch `json:"items"`
}

func init() {
	SchemeBuilder.Register(&SecSynch{}, &SecSynchList{})
}
