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
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TriggerSpec defines the desired state of Trigger.
type TriggerSpec struct {
	Event string `json:"event"`

	// Conditions is a list of conditions that must be met for the trigger to fire.
	Conditions []string `json:"conditions"`

	Actions []TriggerAction `json:"actions"`
}

type TriggerAction struct {
	Action string                   `json:"action"`
	Params []TriggerActionParameter `json:"params"`
}

type TriggerActionParameter struct {
	Name     string                            `json:"name"`
	FromData *TriggerActionParameterDataSource `json:"fromData,omitempty"`
	FromYaml *apiextensionsv1.JSON             `json:"fromYaml,omitempty"`
}

type TriggerActionParameterDataSource struct {
	Name string `json:"name"`
}

type TriggerActionParameterJsonSource struct {
	Json string `json:"json"`
}

// TriggerStatus defines the observed state of Trigger.
type TriggerStatus struct {
	History []TriggerHistory `json:"history"`
}

type TriggerHistory struct {
	Time   metav1.Time `json:"time"`
	Source string      `json:"source"`
	ID     string      `json:"id"`

	Message []TriggerHistoryMessage `json:"message"`
}

type TriggerHistoryMessage struct {
	Time    metav1.Time `json:"time"`
	Success bool        `json:"success"`
	Message string      `json:"message"`
}

func (th *TriggerHistory) AddMessage(success bool, message string) {
	th.Message = append(th.Message, TriggerHistoryMessage{
		Time:    metav1.Now(),
		Success: success,
		Message: message,
	})
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:singular="wrgtrigger",shortName={"wrgtrg","wrgtrgs"}

// Trigger is the Schema for the triggers API.
type Trigger struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TriggerSpec   `json:"spec,omitempty"`
	Status TriggerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TriggerList contains a list of Trigger.
type TriggerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Trigger `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Trigger{}, &TriggerList{})
}
