package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func init() {
	SchemeBuilder.Register(&DeployResource{}, &DeployResourceList{})
}

// DeployResourceStatuseSpec contains sync & health status of application's resource
// type DeployResourceStatusSpec struct {
// 	SyncStatus   SyncStatusCode   `json:"syncstatus"`
// 	HealthStatus HealthStatusCode `json:"healthstatus"`
// }

// DeployResourceSpec is a spec of deployed application's resource
type DeployResourceSpec struct {
	//name kind namespace, status
	Name       string `json:"name"`
	Namespace  string `json:"namespace"`
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
}

// +kubebuilder:object:root=true

// DeployResource is resource created by an application
// +kubebuilder:resource:path=deployresources,scope=Namespaced,shortName=drs
type DeployResource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Application       string             `json:"application"`
	Spec              DeployResourceSpec `json:"spec"`
}

// +kubebuilder:object:root=true

// DeployResourceList contains the list of DeployResources
type DeployResourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []DeployResource `json:"items"`
}

// SyncStatusCode is a type which represents possible comparison results
// type SyncStatusCode string

// // Possible comparison results
// const (
// 	// SyncStatusCodeUnknown indicates that the status of a sync could not be reliably determined
// 	SyncStatusCodeUnknown SyncStatusCode = "Unknown"
// 	// SyncStatusCodeOutOfSync indicates that desired and live states match
// 	SyncStatusCodeSynced SyncStatusCode = "Synced"
// 	// SyncStatusCodeOutOfSync indicates that there is a drift between desired and live states
// 	SyncStatusCodeOutOfSync SyncStatusCode = "OutOfSync"
// )

// // HealthStatusCode Represents resource health status
// type HealthStatusCode string

// const (
// 	// HealthStatusUnknown indicates that health assessment failed and actual health status is unknown
// 	HealthStatusUnknown HealthStatusCode = "Unknown"
// 	// HealthStatusProgressing indicates that resource is not healthy but still have a chance to reach healthy state
// 	HealthStatusProgressing HealthStatusCode = "Progressing"
// 	// HealthStatusHealthy indicates that resource is 100% healthy
// 	HealthStatusHealthy HealthStatusCode = "Healthy"
// 	// HealthStatusSuspended is assigned to resources that are suspended or paused. The typical example is a
// 	// [suspended](https://kubernetes.io/docs/tasks/job/automated-tasks-with-cron-jobs/#suspend) CronJob.
// 	HealthStatusSuspended HealthStatusCode = "Suspended"
// 	// HealthStatusDegraded status is used if resource status indicates failure or resource could not reach healthy state
// 	// within some timeout.
// 	HealthStatusDegraded HealthStatusCode = "Degraded"
// 	// HealthStatusMissing indicates that resource is missing in the cluster.
// 	HealthStatusMissing HealthStatusCode = "Missing"
// )
