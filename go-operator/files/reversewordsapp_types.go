package v1alpha1

import (
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReverseWordsAppSpec defines the desired state of ReverseWordsApp
// +k8s:openapi-gen=true
type ReverseWordsAppSpec struct {
     Replicas int32  `json:"replicas"`
     AppVersion string `json:"appVersion"`
}

// ReverseWordsAppStatus defines the observed state of ReverseWordsApp
// +k8s:openapi-gen=true
type ReverseWordsAppStatus struct {
    AppPods []string `json:"appPods"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReverseWordsApp is the Schema for the reversewordsapps API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type ReverseWordsApp struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`

    Spec   ReverseWordsAppSpec   `json:"spec,omitempty"`
    Status ReverseWordsAppStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ReverseWordsAppList contains a list of ReverseWordsApp
type ReverseWordsAppList struct {
    metav1.TypeMeta `json:",inline"`
    metav1.ListMeta `json:"metadata,omitempty"`
    Items           []ReverseWordsApp `json:"items"`
}

func init() {
    SchemeBuilder.Register(&ReverseWordsApp{}, &ReverseWordsAppList{})
}
