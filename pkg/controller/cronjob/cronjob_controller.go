package cronjob

import (
	"context"
	"fmt"
	"sort"
	"time"

	cachev1alpha1 "example/cronjob-operator/pkg/apis/cache/v1alpha1"
	kbatch "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	ref "k8s.io/client-go/tools/reference"
	"github.com/robfig/cron"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = logf.Log.WithName("controller_cronjob")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CronJob Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

func Add(mgr manager.Manager) error {

	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCronJob{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("cronjob-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CronJob
	err = c.Watch(&source.Kind{Type: &cachev1alpha1.CronJob{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CronJob
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &cachev1alpha1.CronJob{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that ReconcileCronJob implements reconcile.Reconciler
var _ reconcile.Reconciler = &ReconcileCronJob{}

// ReconcileCronJob reconciles a CronJob object
type ReconcileCronJob struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}


var (
	scheduledTimeAnnotation = "batch.cache.example.com/scheduled-at"
	jobOwnerKey = "metadata-controller"
	//apiGVStr    = ""
)
// Reconcile reads that state of the cluster for a CronJob object and makes changes based on the state read
// and what is in the CronJob.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCronJob) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := context.Background()
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CronJob")

	// Fetch the CronJob instance
	cronJob := cachev1alpha1.CronJob{}
	err := r.client.Get(ctx, request.NamespacedName, &cronJob)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}


	var childJobs kbatch.JobList
	if err := r.client.List(ctx, &childJobs, client.InNamespace(request.Namespace), client.MatchingLabels{jobOwnerKey:cronJob.Name}); err != nil {
		log.Error(err, "unable to list child Jobs")
		return reconcile.Result{}, err
	}
	// find the active list of jobs
	var activeJobs []*kbatch.Job
	var successfulJobs []*kbatch.Job
	var failedJobs []*kbatch.Job
	var mostRecentTime *time.Time // find the last run so we can update the status


	isJobFinished := func(job *kbatch.Job) (bool, kbatch.JobConditionType) {
		for _, c := range job.Status.Conditions {
			if (c.Type == kbatch.JobComplete || c.Type == kbatch.JobFailed) && c.Status == corev1.ConditionTrue {
				return true, c.Type
			}
		}

		return false, ""
	}

	getScheduledTimeForJob := func(job *kbatch.Job) (*time.Time, error) {
		timeRaw := job.Annotations[scheduledTimeAnnotation]
		if len(timeRaw) == 0 {
			return nil, nil
		}

		timeParsed, err := time.Parse(time.RFC3339, timeRaw)
		if err != nil {
			return nil, err
		}
		return &timeParsed, nil
	}
	for i, job := range childJobs.Items {
		_, finishedType := isJobFinished(&job)
		switch finishedType {
		case "": // ongoing
			activeJobs = append(activeJobs, &childJobs.Items[i])
		case kbatch.JobFailed:
			failedJobs = append(failedJobs, &childJobs.Items[i])
		case kbatch.JobComplete:
			successfulJobs = append(successfulJobs, &childJobs.Items[i])
		}

		// We'll store the launch time in an annotation, so we'll reconstitute that from
		// the active jobs themselves.
		scheduledTimeForJob, err := getScheduledTimeForJob(&job)
		if err != nil {
			log.Error(err, "unable to parse schedule time for child job", "job", &job)
			continue
		}
		if scheduledTimeForJob != nil {
			if mostRecentTime == nil {
				mostRecentTime = scheduledTimeForJob
			} else if mostRecentTime.Before(*scheduledTimeForJob) {
				mostRecentTime = scheduledTimeForJob
			}
		}
	}

	if mostRecentTime != nil {
		cronJob.Status.LastScheduleTime = &metav1.Time{Time: *mostRecentTime}
	} else {
		cronJob.Status.LastScheduleTime = nil
	}
	cronJob.Status.Active = nil
	for _, activeJob := range activeJobs {
		jobRef, err := ref.GetReference(r.scheme, activeJob)
		if err != nil {
			log.Error(err, "unable to make reference to active job", "job", activeJob)
			continue
		}
		cronJob.Status.Active = append(cronJob.Status.Active, *jobRef)
	}
	log.V(1).Info("job count", "active jobs", len(activeJobs), "successful jobs", len(successfulJobs), "failed jobs", len(failedJobs))

	if err := r.client.Status().Update(ctx, &cronJob); err != nil {
		log.Error(err, "unable to update CronJob status")
		return reconcile.Result{}, err
	}


	if cronJob.Spec.FailedJobsHistoryLimit != nil {
		sort.Slice(failedJobs, func(i, j int) bool {
			if failedJobs[i].Status.StartTime == nil {
				return failedJobs[j].Status.StartTime != nil
			}
			return failedJobs[i].Status.StartTime.Before(failedJobs[j].Status.StartTime)
		})
		for i, job := range failedJobs {
			if int32(i) >= int32(len(failedJobs))-*cronJob.Spec.FailedJobsHistoryLimit {
				break
			}
			if err := r.client.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); ignoreNotFound(err) != nil {
				log.Error(err, "unable to delete old failed job", "job", job)
			} else {
				log.V(0).Info("deleted old failed job", "job", job)
			}
		}
	}

	if cronJob.Spec.SuccessfulJobsHistoryLimit != nil {
		sort.Slice(successfulJobs, func(i, j int) bool {
			if successfulJobs[i].Status.StartTime == nil {
				return successfulJobs[j].Status.StartTime != nil
			}
			return successfulJobs[i].Status.StartTime.Before(successfulJobs[j].Status.StartTime)
		})
		for i, job := range successfulJobs {
			if int32(i) >= int32(len(successfulJobs))-*cronJob.Spec.SuccessfulJobsHistoryLimit {
				break
			}
			if err := r.client.Delete(ctx, job, client.PropagationPolicy(metav1.DeletePropagationBackground)); ignoreNotFound(err) != nil {
				log.Error(err, "unable to delete old successful job", "job", job)
			} else {
				log.V(0).Info("deleted old successful job", "job", job)
			}
		}
	}

	if cronJob.Spec.Suspend != nil && *cronJob.Spec.Suspend {
		log.V(1).Info("cronjob suspended, skipping")
		return reconcile.Result{}, nil
	}


	// 检查是否超过了次数
	if cronJob.Spec.ScheduledJobsLimits != nil && cronJob.Status.ScheduledJobsTimes >= *cronJob.Spec.ScheduledJobsLimits {
		log.V(1).Info("执行已经到达最大次数")
		return reconcile.Result{}, nil
	}

	getNextSchedule := func(cronJob *cachev1alpha1.CronJob, now time.Time) (lastMissed time.Time, next time.Time, err error) {
		sched, err := cron.ParseStandard(cronJob.Spec.Schedule)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("Unparseable schedule %q: %v", cronJob.Spec.Schedule, err)
		}

		// for optimization purposes, cheat a bit and start from our last observed run time
		// we could reconstitute this here, but there's not much point, since we've
		// just updated it.
		var earliestTime time.Time
		if cronJob.Status.LastScheduleTime != nil {
			earliestTime = cronJob.Status.LastScheduleTime.Time
		} else {
			earliestTime = cronJob.ObjectMeta.CreationTimestamp.Time
		}
		if cronJob.Spec.StartingDeadlineSeconds != nil {
			// controller is not going to schedule anything below this point
			schedulingDeadline := now.Add(-time.Second * time.Duration(*cronJob.Spec.StartingDeadlineSeconds))

			if schedulingDeadline.After(earliestTime) {
				earliestTime = schedulingDeadline
			}
		}
		if earliestTime.After(now) {
			return time.Time{}, sched.Next(now), nil
		}

		starts := 0
		for t := sched.Next(earliestTime); !t.After(now); t = sched.Next(t) {
			lastMissed = t
			starts++
			if starts > 100 {
				// We can't get the most recent times so just return an empty slice
				return time.Time{}, time.Time{}, fmt.Errorf("Too many missed start times (> 100). Set or decrease .spec.startingDeadlineSeconds or check clock skew.")
			}
		}
		return lastMissed, sched.Next(now), nil
	}

	missedRun, nextRun, err := getNextSchedule(&cronJob, time.Now())
	if err != nil {
		log.Error(err, "unable to figure out CronJob schedule")
		// we don't really care about requeuing until we get an update that
		// fixes the schedule, so don't return an error
		return reconcile.Result{}, nil
	}

	scheduledResult := reconcile.Result{RequeueAfter: nextRun.Sub(time.Now())} // save this so we can re-use it elsewhere
	log = log.WithValues("now", time.Now(), "next run", nextRun)
	if missedRun.IsZero() {
		log.V(1).Info("no upcoming scheduled times, sleeping until next")
		return scheduledResult, nil
	}

	log = log.WithValues("current run", missedRun)
	tooLate := false
	if cronJob.Spec.StartingDeadlineSeconds != nil {
		tooLate = missedRun.Add(time.Duration(*cronJob.Spec.StartingDeadlineSeconds) * time.Second).Before(time.Now())
	}
	if tooLate {
		log.V(1).Info("missed starting deadline for last run, sleeping till next")
		// TODO(directxman12): events
		return scheduledResult, nil
	}
	if cronJob.Spec.ConcurrencyPolicy == cachev1alpha1.ForbidConcurrent && len(activeJobs) > 0 {
		log.V(1).Info("concurrency policy blocks concurrent runs, skipping", "num active", len(activeJobs))
		return scheduledResult, nil
	}
	if cronJob.Spec.ConcurrencyPolicy == cachev1alpha1.ReplaceConcurrent {
		for _, activeJob := range activeJobs {
			// we don't care if the job was already deleted
			if err := r.client.Delete(ctx, activeJob, client.PropagationPolicy(metav1.DeletePropagationBackground)); ignoreNotFound(err) != nil {
				log.Error(err, "unable to delete active job", "job", activeJob)
				return reconcile.Result{}, err
			}
		}
	}
	constructJobForCronJob := func(cronJob *cachev1alpha1.CronJob, scheduledTime time.Time) (*kbatch.Job, error) {
		// We want job names for a given nominal start time to have a deterministic name to avoid the same job being created twice
		name := fmt.Sprintf("%s-%d", cronJob.Name, scheduledTime.Unix())

		job := &kbatch.Job{
			ObjectMeta: metav1.ObjectMeta{
				Labels:      make(map[string]string),
				Annotations: make(map[string]string),
				Name:        name,
				Namespace:   cronJob.Namespace,
			},
			Spec: *cronJob.Spec.JobTemplate.Spec.DeepCopy(),
		}
		for k, v := range cronJob.Spec.JobTemplate.Annotations {
			job.Annotations[k] = v
		}
		job.Annotations[scheduledTimeAnnotation] = scheduledTime.Format(time.RFC3339)
		for k, v := range cronJob.Spec.JobTemplate.Labels {
			job.Labels[k] = v
		}
		job.Labels[jobOwnerKey] = cronJob.Name;
		if err := ctrl.SetControllerReference(cronJob, job, r.scheme); err != nil {
			return nil, err
		}

		return job, nil
	}

	job, err := constructJobForCronJob(&cronJob, missedRun)
	if err != nil {
		log.Error(err, "unable to construct job from template")
		// don't bother requeuing until we get a change to the spec
		return scheduledResult, nil
	}

	// ...and create it on the cluster
	if err := r.client.Create(ctx, job); err != nil {
		log.Error(err, "unable to create Job for CronJob", "job", job)
		return reconcile.Result{}, err
	}
	cronJob.Status.ScheduledJobsTimes += 1
	if err := r.client.Status().Update(ctx, &cronJob); err != nil {
		return reconcile.Result{}, err
	}
	log.V(1).Info("created Job for CronJob run", "job", job)
	// Define a new Pod object
	return scheduledResult, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
//func newPodForCR(cr *cachev1alpha1.CronJob) *corev1.Pod {
//	labels := map[string]string{
//		"app": cr.Name,
//	}
//	return &corev1.Pod{
//		ObjectMeta: metav1.ObjectMeta{
//			Name:      cr.Name + "-pod",
//			Namespace: cr.Namespace,
//			Labels:    labels,
//		},
//		Spec: corev1.PodSpec{
//			Containers: []corev1.Container{
//				{
//					Name:    "busybox",
//					Image:   "busybox",
//					Command: []string{"sleep", "3600"},
//				},
//			},
//		},
//	}
//}

