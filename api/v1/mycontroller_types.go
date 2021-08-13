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

package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// MyControllerSpec defines the desired state of MyController
type MyControllerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of MyController. Edit mycontroller_types.go to remove/update
	// Foo              string `json:"foo,omitempty"`

	Secret           string              `json:"secret"`
	DisableAnalytics bool                `json:"disable_analytics"`
	LogLevel         string              `json:"log_level"`
	Storage          MyControllerStorage `json:"storage"`
}

type MyControllerStorage struct {
	StorageClassName string             `json:"storage_class_name"`
	SizeData         *resource.Quantity `json:"data_size,omitempty"`
	SizeMetric       *resource.Quantity `json:"metric_size,omitempty"`
}

// MyControllerStatus defines the observed state of MyController
type MyControllerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// MyController is the Schema for the mycontrollers API
type MyController struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MyControllerSpec   `json:"spec,omitempty"`
	Status MyControllerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MyControllerList contains a list of MyController
type MyControllerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MyController `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MyController{}, &MyControllerList{})
}
