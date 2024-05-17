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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Represents the possible execution states of a ClusterScan.
const (
	ExecutionStatusSuccess         = "Success"
	ExecutionStatusFailure         = "Failure"
	ExecutionStatusInProgress      = "InProgress"
	ExecutionStatusInProgressRetry = "InProgress-Retry"
	ExecutionStatusInit            = "Initialized"
)

// Defines the targets for a ClusterScan.
type ClusterScanTargets struct {
	Namespaces []string          `json:"namespaces,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// ClusterScanSpec defines the desired state of ClusterScan
type ClusterScanSpec struct {
	// Type of job to execute
	JobType string `json:"jobType"`
	// A cron scan schedule.
	Schedule string `json:"schedule,omitempty"`
	//Job-specific configuration options.
	JobParams map[string]string `json:"jobParams,omitempty"`
	// Allow for concurrent scans. (Defualt: False)
	AllowConcurrency bool `json:"allowConcurrency,omitempty"`
	// Maximum number of retries for failed scans. (Default: 0)
	MaxRetry int `json:"maxRetry,omitempty"`
	// Namespaces and labels that the scan should target.
	Targets ClusterScanTargets `json:"targets,omitempty"`
	// Indicated the ClusterScan (Reconciliation) is stopped.
	Stopped bool `json:"active,omitempty"`
}

// ClusterScanStatus defines the observed state of ClusterScan
type ClusterScanStatus struct {
	// Timestamp of the last status change.
	LastStatusChange metav1.Time `json:"LastStatusChange,omitempty"`
	// Status of the last scan execution (Success, Failure, InProgress).
	ExecutionStatus string `json:"executionStatus,omitempty"`
	// Results contains a summary of the scan results.
	Results string `json:"results,omitempty"`
	// Number of resources has processed.
	// ScannedResources int `json:"scannedResources,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// ClusterScan is the Schema for the clusterscans API
type ClusterScan struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ClusterScanSpec   `json:"spec,omitempty"`
	Status ClusterScanStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ClusterScanList contains a list of ClusterScan
type ClusterScanList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ClusterScan `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ClusterScan{}, &ClusterScanList{})
}
