/*
Copyright 2022 Alex.

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

type UbuntuKernelModuleSpec struct {
	DesiredModules []string `json:"desiredModules"`
}

type Node struct {
	Name    string   `json:"name"`
	Modules []Module `json:"modules"`
}

type Module struct {
	Name   string `json:"name,omitempty"`
	UsedBy string `json:"usedBy,omitempty"`
	Size   string `json:"size,omitempty"`
}

// UbuntuKernelModuleStatus defines the observed state of UbuntuKernelModule
type UbuntuKernelModuleStatus struct {
	Nodes []Node `json:"nodes"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// UbuntuKernelModule is the Schema for the ubuntukernelmodules API
type UbuntuKernelModule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   UbuntuKernelModuleSpec   `json:"spec,omitempty"`
	Status UbuntuKernelModuleStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// UbuntuKernelModuleList contains a list of UbuntuKernelModule
type UbuntuKernelModuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []UbuntuKernelModule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&UbuntuKernelModule{}, &UbuntuKernelModuleList{})
}
