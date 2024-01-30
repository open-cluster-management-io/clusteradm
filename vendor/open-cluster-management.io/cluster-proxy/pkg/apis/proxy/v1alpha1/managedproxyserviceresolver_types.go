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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&ManagedProxyServiceResolver{}, &ManagedProxyServiceResolverList{})
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster

// +genclient
// +genclient:nonNamespaced
// ManagedProxyServiceResolver defines a target service that need to expose from a set of managed clusters to the hub.
// To access a target service on a managed cluster from hub. First, users need to apply a proper ManagedProxyServiceResolver.
// The managed cluster should match the ManagedClusterSet in the ManagedProxyServiceResolver.Spec. The serviceNamespace and serviceName should also match the target service.
// A usage example: /examples/access-other-services/main.go
type ManagedProxyServiceResolver struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManagedProxyServiceResolverSpec   `json:"spec,omitempty"`
	Status ManagedProxyServiceResolverStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManagedProxyServiceResolverList contains a list of ManagedProxyServiceResolver
type ManagedProxyServiceResolverList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManagedProxyServiceResolver `json:"items"`
}

// ManagedProxyServiceResolverSpec defines the desired state of ManagedProxyServiceResolver.
type ManagedProxyServiceResolverSpec struct {
	// ManagedClusterSelector selects a set of managed clusters.
	// +required
	ManagedClusterSelector ManagedClusterSelector `json:"managedClusterSelector"`

	// ServiceSelector selects a service.
	// +required
	ServiceSelector ServiceSelector `json:"serviceSelector"`
}

// ManagedClusterSelectorType is the type of ManagedClusterSelector.
// +kubebuilder:validation:Enum=ManagedClusterSet
type ManagedClusterSelectorType string

var (
	// ManagedClusterSetSelectorType indicates the selector is a ManagedClusterSet.
	// In this type, the manageclusterset field of the selector is required.
	ManagedClusterSelectorTypeClusterSet ManagedClusterSelectorType = "ManagedClusterSet"
)

type ManagedClusterSelector struct {
	// Type represents the type of the selector. Now only ManagedClusterSet is supported.
	// +optional
	// +kubebuilder:default=ManagedClusterSet
	Type ManagedClusterSelectorType `json:"type,omitempty"`

	// ManagedClusterSet defines a set of managed clusters that need to expose the service.
	// +optional
	ManagedClusterSet *ManagedClusterSet `json:"managedClusterSet,omitempty"`
}

// ManagedClusterSet defines the name of a managed cluster set.
type ManagedClusterSet struct {
	// Name is the name of the managed cluster set.
	// +required
	Name string `json:"name"`
}

// ServiceSelectorType is the type of ServiceSelector.
// +kubebuilder:validation:Enum=ServiceRef
type ServiceSelectorType string

var (
	// ServiceSelectorTypeServiceRef indicates the selector requires serviceNamespace and serviceName fields.
	ServiceSelectorTypeServiceRef ServiceSelectorType = "ServiceRef"
)

type ServiceSelector struct {
	// Type represents the type of the selector. Now only ServiceRef type is supported.
	// +optional
	// +kubebuilder:default=ServiceRef
	Type ServiceSelectorType `json:"type,omitempty"`

	// ServiceRef defines a service in a namespace.
	// +optional
	ServiceRef *ServiceRef `json:"serviceRef,omitempty"`
}

// ServiceRef represents a service in a namespace.
type ServiceRef struct {
	// Namespace represents the namespace of the service.
	// +required
	Namespace string `json:"namespace"`

	// Name represents the name of the service.
	// +required
	Name string `json:"name"`
}

// ManagedProxyServiceResolverStatus defines the observed state of ManagedProxyServiceResolver.
type ManagedProxyServiceResolverStatus struct {
	// Conditions contains the different condition statuses for this ManagedProxyServiceResolver.
	Conditions []metav1.Condition `json:"conditions"`
}

const (
	ConditionTypeServiceResolverAvaliable = "ServiceResolverAvaliable"
)
