package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type ConcurrencyPolicy string

const (
	AllowConcurrent   ConcurrencyPolicy = "Allow"
	ForbidConcurrent  ConcurrencyPolicy = "Forbid"
	ReplaceConcurrent ConcurrencyPolicy = "Replace"
)

// CronJobSpec defines the desired state of CronJob
// +k8s:openapi-gen=true
type CronJobSpec struct {

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html


	// +kubebuilder:validation:MinLength=0
	Schedule string `json:"schedule"`

	// +kubebuilder:validation:Minimum=0
	// +optional
	StartingDeadlineSeconds *int64 `json:"startingDeadlineSeconds,omitempty"`

	// +kubebuilder:validation:Enum=Allow;Forbid;Replace
	// +optional
	ConcurrencyPolicy ConcurrencyPolicy `json:"concurrencyPolicy,omitempty"`

	// +optional
	Suspend *bool `json:"suspend,omitempty"`

	JobTemplate batchv1beta1.JobTemplateSpec `json:"jobTemplate"`

	// +kubebuilder:validation:Minimum=0
	// +optional
	SuccessfulJobsHistoryLimit *int32 `json:"successfulJobsHistoryLimit,omitempty"`

	// +kubebuilder:validation:Minimum=0
	// +optional
	FailedJobsHistoryLimit *int32 `json:"failedJobsHistoryLimit,omitempty"`


    ScheduledJobsLimits *int32 `json:"scheduledJobsLimits,omitempty"`

}

// CronJobStatus defines the observed state of CronJob
// +k8s:openapi-gen=true
type CronJobStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book.kubebuilder.io/beyond_basics/generating_crd.html

	// +optional
	Active []corev1.ObjectReference `json:"active,omitempty"`

	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// +optional
	ScheduledJobsTimes int32 `json:"executeTimes,omitempty"`

}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronJob is the Schema for the cronjobs API
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type CronJob struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronJobSpec   `json:"spec,omitempty"`
	Status CronJobStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronJobList contains a list of CronJob
type CronJobList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronJob `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CronJob{}, &CronJobList{})
}
