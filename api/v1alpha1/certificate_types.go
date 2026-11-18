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

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CertificateSpec defines the desired state of Certificate
type CertificateSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	// SecretName appoints the Secret of certificate and private key.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=253
	// +kubebuilder:validation:XValidation:rule="self == oldSelf",message="secretName is immutable"
	SecretName string `json:"secretName"`

	// DNSNames sets the domains included in the certificate.
	// +kubebuilder:validation:MinItems=1
	// +kubebuilder:validation:MaxItems=100
	DNSNames []string `json:"dnsNames"`

	// Duration sets the valid range of the certificate.
	// +kubebuilder:default="24h"
	Duration string `json:"duration,omitempty"`

	// RenewBefore sets the when to renew the certificate before expiring.
	// +kubebuilder:default="8h"
	RenewBefore string `json:"renewBefore,omitempty"`
}

// CertificateStatus defines the observed state of Certificate.
type CertificateStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// ObservedGeneration indicates the version which has been processsed by the controller.
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// NotAfter indicates the expiration time.
	NotAfter *metav1.Time `json:"notAfter,omitempty"`

	// RenewalTime indicates the renewal tiem.
	RenewalTime *metav1.Time `json:"renewalTime,omitempty"`

	// conditions represent the current state of the Certificate resource.
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
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// Certificate is the Schema for the certificates API
type Certificate struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of Certificate
	// +required
	Spec CertificateSpec `json:"spec"`

	// status defines the observed state of Certificate
	// +optional
	Status CertificateStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// CertificateList contains a list of Certificate
type CertificateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []Certificate `json:"items"`
}

func init() {
	SchemeBuilder.Register(func(s *runtime.Scheme) error {
		s.AddKnownTypes(SchemeGroupVersion, &Certificate{}, &CertificateList{})
		return nil
	})
}
