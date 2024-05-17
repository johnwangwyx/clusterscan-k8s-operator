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

package controller

import (
	"context"
	"fmt"
	"sort"
	"time"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1" // Import corev1
	errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	johnwangwyxiov1 "github.com/johnwangwyx/clusterscan-k8s-operator/api/v1"
	v1 "github.com/johnwangwyx/clusterscan-k8s-operator/api/v1"
	"github.com/robfig/cron/v3"
)

// ClusterScanReconciler reconciles a ClusterScan object
type ClusterScanReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// Map of allowed job types to their Docker images
// Note this for more flexibility, we should use config files, kv store like Consul, or ConfigMap to allow dynamic config changes.
var allowedJobImages = map[string]string{
	"ansible": "johnwangwyx/ansible-job-demo:latest",
}

// Configurable vars. (Again, best fetched from some kind of config management file or system)
const WaitTimeout = 5 * time.Minute
const JobNamePostfix = "-job"
const CronJobNamePostfix = "-cronjob"
const CreateJobWaitDurationSeconds = 5
const CronChildJobLable = "from_cron"

//+kubebuilder:rbac:groups=johnwangwyx.io.io.github.johnwangwyx,resources=clusterscans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=johnwangwyx.io.io.github.johnwangwyx,resources=clusterscans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=johnwangwyx.io.io.github.johnwangwyx,resources=clusterscans/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the ClusterScan object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *ClusterScanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx).WithValues("clusterscan", req.NamespacedName)

	// Fetch the ClusterScan object
	var clusterScan johnwangwyxiov1.ClusterScan
	if err := r.Get(ctx, req.NamespacedName, &clusterScan); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if clusterScan.Spec.Stopped {
		l.Info("ClusterScan is stopped, skipping reconciliation.")
		return ctrl.Result{}, nil
	}

	var obj client.Object
	jobName := clusterScan.Name + JobNamePostfix
	isCronJob := clusterScan.Spec.Schedule != ""

	if isCronJob {
		obj = &batchv1.CronJob{}
		jobName = clusterScan.Name + CronJobNamePostfix
	} else {
		obj = &batchv1.Job{}
	}

	exists, errGet := r.CheckResourceExists(ctx, obj, jobName, clusterScan.Namespace)
	if errGet != nil {
		return ctrl.Result{}, fmt.Errorf("failed to check if resource exists: %w", errGet)
	} else if exists {
		l.Info("Resource already exists, skip creation", "namespace", clusterScan.Namespace, "name", jobName)
	} else {
		// Resource does not exist, create the resource
		if err := r.createJob(ctx, &clusterScan); err != nil {
			l.Error(err, "Failed to create or update Job/CronJob")
			return ctrl.Result{}, err
		}
		// Initialize status and update the LastStatusChange time when creating the job
		clusterScan.Status.ExecutionStatus = v1.ExecutionStatusInit
		clusterScan.Status.LastStatusChange = metav1.Now()
		if err := r.Status().Update(ctx, &clusterScan); err != nil {
			l.Error(err, "Failed to update ClusterScan status after creating job")
			return ctrl.Result{}, err
		}

		l.Info("Initialized Job status", "ExecutionStatus", v1.ExecutionStatusInit, "LastStatusChange", clusterScan.Status.LastStatusChange)
		// Requeue shortly afte creation to allow the job to initialize
		return ctrl.Result{RequeueAfter: CreateJobWaitDurationSeconds * time.Second}, nil
	}

	l.Info("Looking for possible reconciliation", "namespace", clusterScan.Namespace, "name", jobName)
	var job *batchv1.Job
	var err error
	if !isCronJob {
		// One-off Job
		job = &batchv1.Job{}
		l.Info("Getting the job", "namespace", clusterScan.Namespace, "name", jobName)
		err = r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: clusterScan.Namespace}, job)
		if err != nil && !errors.IsNotFound(err) {
			return ctrl.Result{}, err
		}
	} else {
		// CronJob - get the most recent job created by the CronJob
		l.Info("Getting the most recent cron children job", "namespace", clusterScan.Namespace, "name", jobName)
		job, err = r.getMostRecentJob(ctx, clusterScan)
		if err != nil {
			return ctrl.Result{}, err
		}
	}

	// Check if job is nil and just initialized
	if job == nil && clusterScan.Status.ExecutionStatus == v1.ExecutionStatusInit {
		timeSinceLastStatusChange := time.Since(clusterScan.Status.LastStatusChange.Time)
		if timeSinceLastStatusChange < WaitTimeout {
			l.Info("Waiting for the job to be initialized", "timeRemaining", WaitTimeout-timeSinceLastStatusChange)
			return ctrl.Result{RequeueAfter: CreateJobWaitDurationSeconds * time.Second}, nil
		} else {
			l.Error(fmt.Errorf("job not found and status has been 'Init' for more than the required limit"), "Failing reconciliation", "jobName", jobName, "WaitTimeout", WaitTimeout)
			// Updating the status to a failure
			clusterScan.Status.ExecutionStatus = v1.ExecutionStatusFailure
			clusterScan.Status.LastStatusChange = metav1.Now()
			if err := r.Status().Update(ctx, &clusterScan); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed to update ClusterScan status to error: %w", err)
			}
			return ctrl.Result{}, fmt.Errorf("job not found and status has been 'Init' for over 5 minutes")
		}
	}

	currentStatus := clusterScan.Status.ExecutionStatus
	l.Info("Pulling for status", "currentStatus", currentStatus)
	newStatus := ""
	switch {
	case job.Status.Succeeded > 0:
		if isCronJob { // Cronjob runs periodically, so it is still in progress
			newStatus = v1.ExecutionStatusInProgress
		} else { // Report Job success
			newStatus = v1.ExecutionStatusSuccess
		}
	case job.Status.Failed > 0:
		if job.Status.Failed < int32(clusterScan.Spec.MaxRetry) { // if custom max retry limit is not reached
			newStatus = v1.ExecutionStatusInProgressRetry
		} else { // Failed count exceed max retry, reporting status failure
			newStatus = v1.ExecutionStatusFailure
		}
	default: // Not Failing and not Succeeding, it is in-progress
		newStatus = v1.ExecutionStatusInProgress
	}

	// Update status and last status change time only if there's a change
	if currentStatus != newStatus {
		l.Info("Updating status", "currentStatus", currentStatus, "newStatus", newStatus)
		clusterScan.Status.ExecutionStatus = newStatus
		clusterScan.Status.LastStatusChange = metav1.Now()
		if err := r.Status().Update(ctx, &clusterScan); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to update ClusterScan status: %w", err)
		}
		l.Info("Status updated successfully", "updatedStatus", newStatus, "LastStatusChange", clusterScan.Status.LastStatusChange)
	} else {
		l.Info("Status unchanged, no update required", "status", currentStatus)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterScanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&johnwangwyxiov1.ClusterScan{}).
		Owns(&batchv1.Job{}).
		Complete(r)
}

// CheckResourceExists checks if a Kubernetes resource already exists.
func (r *ClusterScanReconciler) CheckResourceExists(ctx context.Context, obj client.Object, name string, namespace string) (bool, error) {
	key := types.NamespacedName{Name: name, Namespace: namespace}
	err := r.Get(ctx, key, obj)
	if err == nil {
		return true, nil // Resource exists
	} else if errors.IsNotFound(err) {
		return false, nil // Resource does not exist
	} else {
		return false, err // An error occurred
	}
}

func (r *ClusterScanReconciler) getMostRecentJob(ctx context.Context, clusterScan johnwangwyxiov1.ClusterScan) (*batchv1.Job, error) {
	l := log.FromContext(ctx).WithValues("ClusterScan", clusterScan.Name, "namespace", clusterScan.Namespace)
	l.Info("Fetching most recent job for CronJob")

	var jobList batchv1.JobList

	// Fetch the children job from the cron job with custom lable
	labelSelector := client.MatchingLabels{"from_cron": clusterScan.Name + CronJobNamePostfix}
	if err := r.List(ctx, &jobList, client.InNamespace(clusterScan.Namespace), labelSelector); err != nil {
		return nil, fmt.Errorf("failed to list jobs for CronJob %s: %w", clusterScan.Name, err)
	}

	l.Info("Jobs retrieved", "jobCount", len(jobList.Items))
	// Sort jobs by creation timestamp, newest first
	sort.Slice(jobList.Items, func(i, j int) bool {
		return jobList.Items[i].CreationTimestamp.After(jobList.Items[j].CreationTimestamp.Time)
	})

	if len(jobList.Items) == 0 {
		l.Info("No jobs found for CronJob")
		return nil, nil // No jobs found
	}

	mostRecentJob := &jobList.Items[0]
	l.Info("Most recent job selected", "jobName", mostRecentJob.Name, "creationTime", mostRecentJob.CreationTimestamp)
	// Return the most recent job if available
	return mostRecentJob, nil
}

// createJob creates a Job or CronJob based on the ClusterScan spec.
func (r *ClusterScanReconciler) createJob(ctx context.Context, clusterScan *johnwangwyxiov1.ClusterScan) error {
	l := log.FromContext(ctx).WithValues("clusterscan", clusterScan.Name)

	l.Info("Validating job type")
	if clusterScan.Spec.JobType == "" {
		return fmt.Errorf("jobType cannot be empty")
	}

	l.Info("Fetching image for job type", "JobType", clusterScan.Spec.JobType)
	// Fetch the image for the specified JobType from the global map
	image, ok := allowedJobImages[clusterScan.Spec.JobType]
	if !ok {
		return fmt.Errorf("unknown job type: %s", clusterScan.Spec.JobType)
	}

	l.Info("Setting restart policy", "MaxRetry", clusterScan.Spec.MaxRetry)
	// Determine the appropriate restart policy
	restartPolicy := corev1.RestartPolicy("OnFailure") // Default to OnFailure if MaxRetry > 0
	if clusterScan.Spec.MaxRetry == 0 {
		restartPolicy = corev1.RestartPolicy("Never") // Set to Never if MaxRetry is 0
	}

	// Pod Template (Reduce loop nesting)
	podTemplate := corev1.PodTemplateSpec{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{"clusterscan": clusterScan.Name},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: restartPolicy,
			Containers: []corev1.Container{
				{
					Name:  clusterScan.Spec.JobType,
					Image: image,
				},
			},
		},
	}

	l.Info("Creating environment variables for the job")
	// Create EnvVars from JobParams
	var envVars []corev1.EnvVar
	for key, value := range clusterScan.Spec.JobParams {
		envVars = append(envVars, corev1.EnvVar{
			Name:  key,
			Value: value,
		})
	}
	podTemplate.Spec.Containers[0].Env = envVars // Set the environment variables

	if clusterScan.Spec.Schedule == "" {
		l.Info("Creating one-off job")
		job := &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterScan.Name + JobNamePostfix,
				Namespace: clusterScan.Namespace,
				OwnerReferences: []metav1.OwnerReference{
					*metav1.NewControllerRef(clusterScan, johnwangwyxiov1.GroupVersion.WithKind(clusterScan.TypeMeta.Kind)),
				},
			},
			Spec: batchv1.JobSpec{
				Template:     podTemplate,
				BackoffLimit: pointer.Int32(int32(clusterScan.Spec.MaxRetry)),
			},
		}
		if err := r.Create(ctx, job); err != nil {
			return fmt.Errorf("failed to create Job: %w", err)
		}

		l.Info("Created Job", "job.Namespace", job.Namespace, "job.Name", job.Name)
	} else {
		// Recurring CronJob
		l.Info("Creating recurring CronJob")
		// Validate Cron Expression
		if _, err := cron.ParseStandard(clusterScan.Spec.Schedule); err != nil {
			return fmt.Errorf("invalid cron expression: %w", err)
		}
		cronJobName := clusterScan.Name + CronJobNamePostfix
		cronChildLabel := map[string]string{
			CronChildJobLable: cronJobName,
		}
		cronJob := &batchv1.CronJob{
			ObjectMeta: metav1.ObjectMeta{
				Name:      cronJobName,
				Namespace: clusterScan.Namespace,
			},
			Spec: batchv1.CronJobSpec{
				Schedule: clusterScan.Spec.Schedule,
				JobTemplate: batchv1.JobTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: cronChildLabel,
						OwnerReferences: []metav1.OwnerReference{
							*metav1.NewControllerRef(clusterScan, johnwangwyxiov1.GroupVersion.WithKind(clusterScan.TypeMeta.Kind)),
						},
					},
					Spec: batchv1.JobSpec{
						Template:     podTemplate,
						BackoffLimit: pointer.Int32(int32(clusterScan.Spec.MaxRetry)),
					},
				},
			},
		}
		if err := r.Create(ctx, cronJob); err != nil {
			return fmt.Errorf("failed to create CronJob: %w", err)
		}
		l.Info("Created CronJob", "cronjob.Namespace", cronJob.Namespace, "cronjob.Name", cronJob.Name)
	}
	return nil
}
