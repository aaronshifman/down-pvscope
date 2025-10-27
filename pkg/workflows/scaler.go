package workflows

import (
	"time"

	proto "github.com/aaronshifman/down-pvscope/api/down-pvscope/v1"
	"github.com/aaronshifman/down-pvscope/pkg/activities"
	"github.com/aaronshifman/down-pvscope/pkg/util"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	corev1 "k8s.io/api/core/v1"
)

const TaskQueueName = "down-pvscope"

// nolint: funlen
func ScaleDownWorkflow(ctx workflow.Context, input *proto.Scale) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting workflow", "namespace", input.Namespace, "newSize", input.Size, "pvcTarget", input.Pvc, "sts", input.Sts)
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			MaximumInterval:    time.Minute,
			BackoffCoefficient: 2,
			MaximumAttempts:    5,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)
	var pvca *activities.PVCActivities
	var pva *activities.PVActivities
	var ja *activities.JobActivities
	var sts *activities.STSActivities

	// get original PVC
	logger.Info("Getting the original PVC", "pvc", input.Pvc, "namespace", input.Namespace)
	originalPVC := util.PvcInfo{}
	err := workflow.ExecuteActivity(ctx, pvca.GetPVC, input.Namespace, input.Pvc).Get(ctx, &originalPVC)
	if err != nil {
		return err
	}
	logger.Debug("Original pvc", "volume", originalPVC.VolumeName, "name", originalPVC.Namespace, "originalStorage", originalPVC.RequestedStorage)

	// mark existing pv safe (retain)
	logger.Info("Marging the original pv retain", "pv", originalPVC.VolumeName)
	var originalRetentionPolicy corev1.PersistentVolumeReclaimPolicy
	err = workflow.ExecuteActivity(ctx, pva.EnsureReclaimPolicyRetain, originalPVC.VolumeName).Get(ctx, &originalRetentionPolicy)
	if err != nil {
		return err
	}
	logger.Debug("pv retention", "original", originalRetentionPolicy)

	// create new PVC / provision new PV
	logger.Info("Provisioning PVC of new size", "newSize", input.Size)
	newPVC := util.PvcInfo{}
	err = workflow.ExecuteActivity(ctx, pvca.CreateStagingPVC, originalPVC, input.Size).Get(ctx, &newPVC)
	if err != nil {
		return err
	}
	logger.Debug("New pvc", "name", newPVC.Name, "size", newPVC.RequestedStorage, "volume", newPVC.VolumeName)

	// make sure new PV is safe
	logger.Info("Ensuring that new PV is retain")
	err = workflow.ExecuteActivity(ctx, pva.EnsureReclaimPolicyRetain, newPVC.VolumeName).Get(ctx, nil)
	if err != nil {
		return err
	}

	// getting initial starting point for replicas
	logger.Info("Getting starting point for replicas", "sts", input.Sts)
	var initialReplicas int64
	err = workflow.ExecuteActivity(ctx, sts.GetInitialReplicase, input.Namespace, input.Sts).Get(ctx, &initialReplicas)
	if err != nil {
		return err
	}
	logger.Debug("Found replicas", "count", initialReplicas)

	// scaling sts to 0
	// TODO: ensure all other pvcs aren't nuked on scale down
	logger.Info("Scaling sts to 0", "sts", input.Sts)
	err = workflow.ExecuteActivity(ctx, sts.ScaleTo0, input.Namespace, input.Sts).Get(ctx, nil)
	if err != nil {
		return err
	}

	// TODO: figure out scale down then rclone or other way around
	logger.Info("Creating RClone job", "originalPVC", originalPVC.Name, "newPVC", newPVC.Name, "originalSize", originalPVC.RequestedStorage, "newSize", newPVC.RequestedStorage)
	err = workflow.ExecuteActivity(ctx, ja.Runrclone, originalPVC, newPVC, input.Namespace).Get(ctx, nil)
	if err != nil {
		return err
	}

	// drop both pvs
	logger.Info("Dropping pvc", "pvc", originalPVC.Name)
	err = workflow.ExecuteActivity(ctx, pvca.DeletePVC, input.Namespace, originalPVC.Name).Get(ctx, nil)
	if err != nil {
		return err
	}

	logger.Info("Dropping pvc", "pvc", newPVC.Name)
	err = workflow.ExecuteActivity(ctx, pvca.DeletePVC, input.Namespace, newPVC.Name).Get(ctx, nil)
	if err != nil {
		return err
	}

	// map the new pv to the original pvc
	logger.Info("Rebinding original PVC name to new PV", "newPV", newPVC.VolumeName, "originalPVC", originalPVC.Name, "newSize", input.Size)
	err = workflow.ExecuteActivity(ctx, pvca.RebindPV, input.Namespace, newPVC.VolumeName, originalPVC, input.Size).Get(ctx, nil)
	if err != nil {
		return err
	}

	logger.Info("Resetting reclaim policy on new PV", "pv", newPVC.VolumeName, "originalPolicy", originalRetentionPolicy)
	err = workflow.ExecuteActivity(ctx, pva.SetReclaimPolicy, newPVC.VolumeName, originalRetentionPolicy).Get(ctx, nil)
	if err != nil {
		return err
	}

	logger.Info("Rescaling sts", "sts", input.Sts)
	err = workflow.ExecuteActivity(ctx, sts.ScaleUp, input.Namespace, input.Sts, initialReplicas).Get(ctx, nil)
	if err != nil {
		return err
	}

	// TODO: optionally drop the original pv

	logger.Info("Workflow done")
	return nil
}
