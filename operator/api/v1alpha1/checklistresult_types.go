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

// Verdict represents the result of a check.
// +kubebuilder:validation:Enum=PASS;WARN;FAIL
type Verdict string

const (
	VerdictPass Verdict = "PASS"
	VerdictWarn Verdict = "WARN"
	VerdictFail Verdict = "FAIL"
)

// ScanType indicates whether this was a full scan or a rescan.
// +kubebuilder:validation:Enum=FullScan;Rescan
type ScanType string

const (
	ScanTypeFull   ScanType = "FullScan"
	ScanTypeRescan ScanType = "Rescan"
)

// ToolCallRecord stores information about a single tool call made by the LLM.
type ToolCallRecord struct {
	// toolName is the name of the tool that was called.
	ToolName string `json:"toolName"`

	// input is the input parameters passed to the tool.
	// +kubebuilder:pruning:PreserveUnknownFields
	Input *runtime.RawExtension `json:"input"`

	// output is the raw response from the tool.
	Output string `json:"output"`
}

// Evidence stores the verification evidence including tool calls.
type Evidence struct {
	// toolCalls contains the tool calls made by the LLM during verification.
	// +optional
	ToolCalls []ToolCallRecord `json:"toolCalls,omitempty"`
}

// CheckResult represents the result of a single check.
type CheckResult struct {
	// id matches the check ID from the ChecklistPolicy.
	ID string `json:"id"`

	// description is the original check description.
	Description string `json:"description"`

	// verdict is the result of the check.
	Verdict Verdict `json:"verdict"`

	// reasoning is the LLM's explanation of the verdict.
	Reasoning string `json:"reasoning"`

	// evidence contains the tool calls and their results.
	// +optional
	Evidence *Evidence `json:"evidence,omitempty"`

	// failedSince is the timestamp when this check first started failing.
	// Used for deduplication of notifications.
	// +optional
	FailedSince *metav1.Time `json:"failedSince,omitempty"`

	// severity from the original check item.
	// +optional
	Severity string `json:"severity,omitempty"`
}

// ChecklistResultSpec defines the desired state of ChecklistResult.
type ChecklistResultSpec struct {
	// policyRef is the name of the ChecklistPolicy that generated this result.
	PolicyRef string `json:"policyRef"`

	// scanType indicates whether this was a full scan or a rescan.
	ScanType ScanType `json:"scanType"`

	// startedAt is when the scan started.
	StartedAt metav1.Time `json:"startedAt"`

	// completedAt is when the scan finished.
	// +optional
	CompletedAt *metav1.Time `json:"completedAt,omitempty"`

	// checks contains the results for each check item.
	// +optional
	Checks []CheckResult `json:"checks,omitempty"`

	// summary contains aggregate counts.
	// +optional
	Summary *ScanSummary `json:"summary,omitempty"`
}

// ChecklistResultStatus defines the observed state of ChecklistResult.
type ChecklistResultStatus struct {
	// phase indicates the current phase of the result.
	// +kubebuilder:validation:Enum=Running;Completed;Failed
	// +optional
	Phase string `json:"phase,omitempty"`

	// conditions represent the current state of the ChecklistResult.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Policy",type=string,JSONPath=`.spec.policyRef`
// +kubebuilder:printcolumn:name="Type",type=string,JSONPath=`.spec.scanType`
// +kubebuilder:printcolumn:name="Pass",type=integer,JSONPath=`.spec.summary.pass`
// +kubebuilder:printcolumn:name="Fail",type=integer,JSONPath=`.spec.summary.fail`
// +kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ChecklistResult is the Schema for the checklistresults API.
type ChecklistResult struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ChecklistResultSpec `json:"spec"`

	// +optional
	Status ChecklistResultStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ChecklistResultList contains a list of ChecklistResult.
type ChecklistResultList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ChecklistResult `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChecklistResult{}, &ChecklistResultList{})
}
