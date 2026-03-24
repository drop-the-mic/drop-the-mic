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
)

// LLMConfig defines the LLM provider configuration.
type LLMConfig struct {
	// provider is the LLM provider to use.
	// +kubebuilder:validation:Enum=claude;gemini;openai
	Provider string `json:"provider"`

	// model is the specific model to use (e.g., "claude-sonnet-4-20250514").
	// +optional
	Model string `json:"model,omitempty"`

	// secretRef references a Secret containing the API key.
	SecretRef SecretReference `json:"secretRef"`
}

// SecretReference references a Kubernetes Secret.
type SecretReference struct {
	// name is the name of the Secret.
	Name string `json:"name"`

	// key is the key within the Secret data.
	// +kubebuilder:default="api-key"
	// +optional
	Key string `json:"key,omitempty"`
}

// CheckItem defines a single check to be performed.
type CheckItem struct {
	// id is a unique identifier for this check within the policy.
	ID string `json:"id"`

	// description is a free-form natural language description of the check.
	// This is passed directly to the LLM without parsing or structuring.
	Description string `json:"description"`

	// severity indicates the importance of this check.
	// +kubebuilder:validation:Enum=critical;warning;info
	// +kubebuilder:default="warning"
	// +optional
	Severity string `json:"severity,omitempty"`
}

// ScheduleConfig defines the dual-loop schedule.
type ScheduleConfig struct {
	// fullScan is the cron expression for full scan schedule.
	FullScan string `json:"fullScan"`

	// failedRescan is the cron expression for failed-only rescan schedule.
	// +optional
	FailedRescan string `json:"failedRescan,omitempty"`
}

// NotificationConfig defines notification channel settings.
type NotificationConfig struct {
	// slack configures Slack notifications.
	// +optional
	Slack *SlackNotification `json:"slack,omitempty"`

	// github configures GitHub Issues notifications.
	// +optional
	GitHub *GitHubNotification `json:"github,omitempty"`

	// jira configures Jira ticket creation.
	// +optional
	Jira *JiraNotification `json:"jira,omitempty"`
}

// SlackNotification defines Slack notification settings.
type SlackNotification struct {
	// channel is the Slack channel to send notifications to.
	Channel string `json:"channel"`

	// secretRef references a Secret containing the Slack webhook URL or token.
	SecretRef SecretReference `json:"secretRef"`
}

// GitHubNotification defines GitHub Issues notification settings.
type GitHubNotification struct {
	// owner is the GitHub repository owner.
	Owner string `json:"owner"`

	// repo is the GitHub repository name.
	Repo string `json:"repo"`

	// labels are labels to apply to created issues.
	// +optional
	Labels []string `json:"labels,omitempty"`

	// secretRef references a Secret containing the GitHub token.
	SecretRef SecretReference `json:"secretRef"`
}

// JiraNotification defines Jira notification settings.
type JiraNotification struct {
	// url is the Jira instance URL.
	URL string `json:"url"`

	// project is the Jira project key.
	Project string `json:"project"`

	// issueType is the Jira issue type.
	// +kubebuilder:default="Bug"
	// +optional
	IssueType string `json:"issueType,omitempty"`

	// secretRef references a Secret containing Jira credentials.
	SecretRef SecretReference `json:"secretRef"`
}

// ChecklistPolicySpec defines the desired state of ChecklistPolicy.
type ChecklistPolicySpec struct {
	// schedule defines the dual-loop scan schedule.
	Schedule ScheduleConfig `json:"schedule"`

	// llm defines the LLM provider configuration.
	LLM LLMConfig `json:"llm"`

	// checks is the list of checks to perform.
	// +kubebuilder:validation:MinItems=1
	Checks []CheckItem `json:"checks"`

	// targetNamespaces limits scanning to specific namespaces.
	// If empty, scans all namespaces the operator has access to.
	// +optional
	TargetNamespaces []string `json:"targetNamespaces,omitempty"`

	// notification defines notification channel settings.
	// +optional
	Notification *NotificationConfig `json:"notification,omitempty"`

	// escalation defines how many consecutive failures trigger escalation.
	// +kubebuilder:default=5
	// +optional
	EscalationThreshold *int32 `json:"escalationThreshold,omitempty"`
}

// ChecklistPolicyStatus defines the observed state of ChecklistPolicy.
type ChecklistPolicyStatus struct {
	// lastFullScanTime is the timestamp of the last full scan.
	// +optional
	LastFullScanTime *metav1.Time `json:"lastFullScanTime,omitempty"`

	// lastRescanTime is the timestamp of the last failed-only rescan.
	// +optional
	LastRescanTime *metav1.Time `json:"lastRescanTime,omitempty"`

	// lastResultName is the name of the most recent ChecklistResult.
	// +optional
	LastResultName string `json:"lastResultName,omitempty"`

	// summary contains aggregate pass/warn/fail counts from the last scan.
	// +optional
	Summary *ScanSummary `json:"summary,omitempty"`

	// conditions represent the current state of the ChecklistPolicy.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// ScanSummary provides aggregate counts from a scan.
type ScanSummary struct {
	// total is the total number of checks.
	Total int32 `json:"total"`

	// pass is the number of checks that passed.
	Pass int32 `json:"pass"`

	// warn is the number of checks with warnings.
	Warn int32 `json:"warn"`

	// fail is the number of checks that failed.
	Fail int32 `json:"fail"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Provider",type=string,JSONPath=`.spec.llm.provider`
// +kubebuilder:printcolumn:name="Checks",type=integer,JSONPath=`.status.summary.total`
// +kubebuilder:printcolumn:name="Pass",type=integer,JSONPath=`.status.summary.pass`
// +kubebuilder:printcolumn:name="Fail",type=integer,JSONPath=`.status.summary.fail`
// +kubebuilder:printcolumn:name="Last Scan",type=date,JSONPath=`.status.lastFullScanTime`
// +kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`

// ChecklistPolicy is the Schema for the checklistpolicies API.
type ChecklistPolicy struct {
	metav1.TypeMeta `json:",inline"`

	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// +required
	Spec ChecklistPolicySpec `json:"spec"`

	// +optional
	Status ChecklistPolicyStatus `json:"status,omitzero"`
}

// +kubebuilder:object:root=true

// ChecklistPolicyList contains a list of ChecklistPolicy.
type ChecklistPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []ChecklistPolicy `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ChecklistPolicy{}, &ChecklistPolicyList{})
}
