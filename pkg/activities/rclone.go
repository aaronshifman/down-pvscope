package activities

import (
	"context"
	"log/slog"
	"time"

	"github.com/aaronshifman/down-pvscope/pkg/util"
	"github.com/pkg/errors"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

type JobActivities struct{}

func (a *JobActivities) Runrclone(ctx context.Context, originalPVC, newPVC *util.PvcInfo, namespace string) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	job := makeJob(originalPVC.Name, newPVC.Name, namespace)
	slog.DebugContext(ctx, "New Job", "name", job.Name, "namespace", job.Namespace)

	// Create the Job in Kubernetes
	jobsClient := client.BatchV1().Jobs(namespace)
	createdJob, err := jobsClient.Create(ctx, job, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "Could not create job")
	}
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 30*time.Minute, true, func(ctx context.Context) (done bool, err error) {
		jobStatus, err := jobsClient.Get(ctx, createdJob.Name, metav1.GetOptions{})
		slog.DebugContext(ctx, "checking job progression", "jobName", createdJob.Name, "success", jobStatus.Status.Succeeded, "failed", jobStatus.Status.Failed)
		if err != nil {
			return false, err
		}

		if jobStatus.Status.Succeeded > 0 {
			return true, nil
		}
		if jobStatus.Status.Failed > 0 {
			return true, nil
		}
		// still running
		// TODO: handle timeouts and kill the job
		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "Unable to complete job successfully")
	}

	// not bothering to wait because the PV/PVC will be bound to the dead pod
	// until it's cleaned up - this is a natural rate limiting
	// TODO: do this properly though
	propagation := metav1.DeletePropagationBackground
	err = jobsClient.Delete(ctx, createdJob.Name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if err != nil {
		return errors.Wrap(err, "Could not delete job")
	}

	return nil
}

func makeJob(src, dst, ns string) *batchv1.Job {
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "rclone-sync-job",
			Namespace: ns,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					// TODO: this is eh-eh-ron hackery for kyverno rewrites
					ImagePullSecrets: []corev1.LocalObjectReference{{Name: "docker-pull-secret"}},
					RestartPolicy:    corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "rclone",
							Image: "rclone/rclone:latest",
							Command: []string{
								"rclone", "sync", "/data/src/", "/data/dest/", "--verbose",
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "source",
									MountPath: "/data/src",
								},
								{
									Name:      "dest",
									MountPath: "/data/dest",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "source",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: src,
								},
							},
						},
						{
							Name: "dest",
							VolumeSource: corev1.VolumeSource{
								PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
									ClaimName: dst,
								},
							},
						},
					},
				},
			},
		},
	}
}
