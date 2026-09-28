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

package controller

import (
	"context"
	"fmt"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	logf "sigs.k8s.io/controller-runtime/pkg/log"

	sregymv1 "yimingsu01/sregym-controller/api/v1"
)

// SREGymExperimentReconciler reconciles a SREGymExperiment object
type SREGymExperimentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	typeAvailableExperiment   = "Available"
	typeProgressingExperiment = "Progressing"
	typeDegradedExperiment    = "Degraded"
)

func (r *SREGymExperimentReconciler) ensureResultsPVC(
	ctx context.Context,
	exp *sregymv1.SREGymExperiment,
	pvcName string,
) error {
	var existingPVC corev1.PersistentVolumeClaim

	err := r.Get(
		ctx,
		types.NamespacedName{
			Name:      pvcName,
			Namespace: exp.Namespace,
		},
		&existingPVC,
	)

	// PVC already exists
	if err == nil {
		return nil
	}

	// Some unexpected error occurred
	if !apierrors.IsNotFound(err) {
		return err
	}

	size := exp.Spec.ResultsStorage.Size
	if size == "" {
		size = "10Gi"
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pvcName,
			Namespace: exp.Namespace,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteMany,
			},

			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: resource.MustParse(size),
				},
			},

			StorageClassName: exp.Spec.ResultsStorage.StorageClassName,
		},
	}
	return r.Create(ctx, pvc)
}

func (r *SREGymExperimentReconciler) jobForProblem(
	experiment *sregymv1.SREGymExperiment,
	jobName string,
	problem string,
	pvcName string,
) *batchv1.Job {

	backoffLimit := int32(2)
	privileged := true
	runAsUser := int64(0)

	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: experiment.Namespace,
		},

		Spec: batchv1.JobSpec{
			BackoffLimit: &backoffLimit,
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "sregym",
							Image: "yimingsu01/sregym-dind:latest",
							SecurityContext: &corev1.SecurityContext{
								Privileged: &privileged,
								RunAsUser:  &runAsUser,
							},

							Args: []string{
								"python",
								"main.py",

								"--problem",
								problem,

								"--agent",
								experiment.Spec.Agent,

								"--model",
								experiment.Spec.Model,

								"--judge-model",
								experiment.Spec.JudgeModel,
							},

							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: experiment.Spec.CredentialsSecretRef.Name,
										},
									},
								},
							},

							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "results",
									MountPath: "/opt/sregym/results",
								},
								{
									Name:      "docker-data",
									MountPath: "/var/lib/docker",
								},
							},
						},
					},

					Volumes: []corev1.Volume{
						{
							Name: "results",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: pvcName,
								},
							},
						},
						{
							Name: "docker-data",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{
									Medium: corev1.StorageMediumDefault,
								},
							},
						},
					},
				},
			},
		},
	}
}

// steps of this controller
// for each of the problem, create a job resource that runs the problem
// in the cluster. create 1 PVC for all jobs in the one CR.
// The job resource should specify the exact commands to run the problem on SREGYm.

// +kubebuilder:rbac:groups=batch.sregym-controller.io,resources=sregymexperiments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch.sregym-controller.io,resources=sregymexperiments/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=batch.sregym-controller.io,resources=sregymexperiments/finalizers,verbs=update
// +kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=batch,resources=jobs/status,verbs=get
// +kubebuilder:rbac:groups="",resources=persistentvolumeclaims,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the SREGymExperiment object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.25.0/pkg/reconcile
func (r *SREGymExperimentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := logf.FromContext(ctx)

	// load the experiment by name
	var exp sregymv1.SREGymExperiment
	if err := r.Get(ctx, req.NamespacedName, &exp); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("Experiment resource not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get CronJob")
		return ctrl.Result{}, err
	}

	// initialize status of the resource
	if len(exp.Status.Conditions) == 0 {
		meta.SetStatusCondition(&exp.Status.Conditions, metav1.Condition{
			Type:    typeProgressingExperiment,
			Status:  metav1.ConditionUnknown,
			Reason:  "Reconciling",
			Message: "Starting reconciliation",
		})
		if err := r.Status().Update(ctx, &exp); err != nil {
			log.Error(err, "Failed to update Experiment status")
			return ctrl.Result{}, err
		}
		if err := r.Get(ctx, req.NamespacedName, &exp); err != nil {
			log.Error(err, "Failed to re-fetch CronJob")
			return ctrl.Result{}, err
		}
	}

	// then, for each of the problems, schedule a job to run them.
	problems := exp.Spec.Problems
	pvcName := exp.Name + "-results"

	if err := r.ensureResultsPVC(ctx, &exp, pvcName); err != nil {
		return ctrl.Result{}, err
	}

	// counters for on going exp
	completed := 0
	failed := 0
	for _, problem := range problems {
		jobName := fmt.Sprintf("%s-problem-%s", exp.Name, problem)

		var existingJob batchv1.Job
		err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: exp.Namespace}, &existingJob)
		switch {
		case err == nil:
			for _, condition := range existingJob.Status.Conditions {
				if condition.Status != corev1.ConditionTrue {
					continue
				}
				switch condition.Type {
				case batchv1.JobComplete:
					completed++
				case batchv1.JobFailed:
					failed++
				}
			}
			continue
		case apierrors.IsNotFound(err):
			// create job
			// if err := r.Get(ctx, types.NamespacedName{Name: jobName, Namespace: exp.Namespace}, &existingJob); err != nil {
			// 	log.Error(err, "Failed to get Job", "problem", problem, "job", jobName)
			// }
			job := r.jobForProblem(
				&exp,
				jobName,
				problem,
				pvcName,
			)

			if err := controllerutil.SetControllerReference(
				&exp,
				job,
				r.Scheme,
			); err != nil {
				return ctrl.Result{}, err
			}

			log.Info(
				"creating job",
				"job", jobName,
				"problem", problem,
			)

			if err := r.Create(ctx, job); err != nil {
				return ctrl.Result{}, err
			}
		default:
			return ctrl.Result{}, err
		}
	}

	before := exp.DeepCopy()

	activeType := typeProgressingExperiment
	reason := "JobsRunning"
	message := fmt.Sprintf("%d/%d Jobs completed; %d failed", completed, len(problems), failed)

	switch {
	case failed > 0:
		activeType = typeDegradedExperiment
		reason = "JobFailed"
	case len(problems) > 0 && completed == len(problems):
		activeType = typeAvailableExperiment
		reason = "AllJobsCompleted"
	}

	changed := false
	for _, conditionType := range []string{
		typeProgressingExperiment,
		typeDegradedExperiment,
		typeAvailableExperiment,
	} {
		status := metav1.ConditionFalse
		if conditionType == activeType {
			status = metav1.ConditionTrue
		}
		if meta.SetStatusCondition(&exp.Status.Conditions, metav1.Condition{
			Type:               conditionType,
			Status:             status,
			Reason:             reason,
			Message:            message,
			ObservedGeneration: exp.Generation,
		}) {
			changed = true
		}
	}
	// this is the real API call to k8s api server
	if changed {
		if err := r.Status().Patch(ctx, &exp, client.MergeFrom(before)); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *SREGymExperimentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&sregymv1.SREGymExperiment{}).
		Owns(&batchv1.Job{}).
		Named("sregymexperiment").
		Complete(r)
}
