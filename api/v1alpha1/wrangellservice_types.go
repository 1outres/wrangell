/*
Copyright 2024.

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

// WrangellServiceSpec defines the desired state of WrangellService.
type WrangellServiceSpec struct {
	// Image is the container image that the service will run
	Image string `json:"image"`

	// Port is the port that the service will listen on
	Port int32 `json:"port"`

	// TargetPort is the port that the service will forward to
	// +optional
	TargetPort int32 `json:"targetPort,omitempty"`

	// IdleTimeout defines the duration (in minutes) a server can remain idle without
	// receiving any requests before it automatically shuts down.
	//+kubebuilder:validation:Minimum=0
	//+optional
	//+kubebuilder:default=180
	IdleTimeout int64 `json:"idleTimeout,omitempty"`
}

// WrangellServiceStatus defines the observed state of WrangellService.
type WrangellServiceStatus struct {
	// Replicas is the number of replicas of the serverless service
	//+kubebuilder:validation:Minimum=0
	//+optional
	Replicas int32 `json:"replicas"`

	// LatestRequest
	//+optional
	LatestRequest metav1.Time `json:"latestRequest,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=".spec.image"
// +kubebuilder:printcolumn:name="Replicas",type="integer",JSONPath=".status.replicas"

// WrangellService is the Schema for the wrangellservices API.
type WrangellService struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   WrangellServiceSpec   `json:"spec,omitempty"`
	Status WrangellServiceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// WrangellServiceList contains a list of WrangellService.
type WrangellServiceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []WrangellService `json:"items"`
}

func init() {
	SchemeBuilder.Register(&WrangellService{}, &WrangellServiceList{})
}
