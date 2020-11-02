/*


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
	"github.com/operator-framework/operator-lib/status"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ReverseWordsAppSpec defines the desired state of ReverseWordsApp
type ReverseWordsAppSpec struct {
	Replicas   int32  `json:"replicas"`
	AppVersion string `json:"appVersion,omitempty"`
}

// ReverseWordsAppStatus defines the observed state of ReverseWordsApp
type ReverseWordsAppStatus struct {
	AppPods    []string          `json:"appPods"`
	Conditions status.Conditions `json:"conditions"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ReverseWordsApp is the Schema for the reversewordsapps API
type ReverseWordsApp struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ReverseWordsAppSpec   `json:"spec,omitempty"`
	Status ReverseWordsAppStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ReverseWordsAppList contains a list of ReverseWordsApp
type ReverseWordsAppList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ReverseWordsApp `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ReverseWordsApp{}, &ReverseWordsAppList{})
}

// Conditions
const (
	// ConditionTypeReverseWordsDeploymentNotReady indicates if the Reverse Words Deployment is not ready
	ConditionTypeReverseWordsDeploymentNotReady status.ConditionType = "ReverseWordsDeploymentNotReady"

	// ConditionTypeReady indicates if the Reverse Words Deployment is ready
	ConditionTypeReady status.ConditionType = "Ready"
)

// SetCondition is the function used to set condition values
func (rwa *ReverseWordsApp) SetCondition(conditionType status.ConditionType, value bool) {
	conditionValue := corev1.ConditionFalse
	if value {
		conditionValue = corev1.ConditionTrue
	}
	condition := status.Condition{
		Type:   conditionType,
		Status: conditionValue,
	}
	rwa.Status.Conditions.SetCondition(condition)
}
