package workflows

import (
	"fmt"
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

	// get original PVC
	originalPVC := util.PvcInfo{}
	err := workflow.ExecuteActivity(ctx, pvca.GetPVC, input.Namespace, input.Pvc).Get(ctx, &originalPVC)
	if err != nil {
		return err
	}
	fmt.Println(originalPVC.VolumeName)

	// mark existing pv safe (retain)
	var originalRetentionPolicy corev1.PersistentVolumeReclaimPolicy
	err = workflow.ExecuteActivity(ctx, pva.EnsureReclaimPolicyRetain, originalPVC.VolumeName).Get(ctx, &originalRetentionPolicy)
	if err != nil {
		return err
	}

	// // create new PVC / provision new PV
	newPVC := util.PvcInfo{}
	err = workflow.ExecuteActivity(ctx, pvca.CreateStagingPVC, originalPVC, input.Size).Get(ctx, &newPVC)
	if err != nil {
		return err
	}
	fmt.Println(originalPVC.VolumeName, newPVC.VolumeName)

	// // make sure new PV is safe
	// // TODO: set the original reclaim policy on this PV
	err = workflow.ExecuteActivity(ctx, pva.EnsureReclaimPolicyRetain, newPVC.VolumeName).Get(ctx, nil)
	if err != nil {
		return err
	}

	fmt.Println("Starting rclone job")
	err = workflow.ExecuteActivity(ctx, ja.Runrclone, originalPVC, newPVC, input.Namespace).Get(ctx, nil)
	if err != nil {
		return err
	}

	// // drop both pvs
	err = workflow.ExecuteActivity(ctx, pvca.DeletePVC, input.Namespace, originalPVC.Name).Get(ctx, nil)
	if err != nil {
		return err
	}
	err = workflow.ExecuteActivity(ctx, pvca.DeletePVC, input.Namespace, newPVC.Name).Get(ctx, nil)
	if err != nil {
		return err
	}

	// map the new pv to the original pvc
	fmt.Println("Rebinding PVC")
	err = workflow.ExecuteActivity(ctx, pvca.RebindPV, input.Namespace, newPVC.VolumeName, originalPVC, input.Size).Get(ctx, nil)
	if err != nil {
		return err
	}

	fmt.Println("Resetting the PV retain policy")
	err = workflow.ExecuteActivity(ctx, pva.SetReclaimPolicy, newPVC.VolumeName, originalRetentionPolicy).Get(ctx, nil)
	if err != nil {
		return err
	}

	// TODO: optionally drop the original pv

	fmt.Println("Done")
	return nil
}
