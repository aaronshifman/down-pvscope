package main

import (
	"context"
	"fmt"
	"time"

	"github.com/aaronshifman/down-pvscope/pkg/activities"
)

func main() {
	ctx := context.Background()

	// find existing
	originalPVC, err := activities.GetPVC(ctx, "down-pvscope", "bogus")
	if err != nil {
		panic(err)
	}
	fmt.Println(originalPVC.Spec.VolumeName)

	// mark existing safe
	// TODO: store original status and restore at end
	_, err = activities.EnsureReclaimPolicyRetain(ctx, originalPVC.Spec.VolumeName)
	if err != nil {
		panic(err)
	}

	// create new PVC / provision new PV
	// waits for pvc to bind before returning
	err = activities.CreateStagingPVC(ctx, "down-pvscope", "bogus", "0.5Gi")
	if err != nil {
		panic(err)
	}

	// find new pv
	newPVC, err := activities.GetPVC(ctx, "down-pvscope", "bogus"+activities.StagingSuffix)
	if err != nil {
		panic(err)
	}
	fmt.Println(newPVC.Spec.VolumeName)

	// make sure new PV is safe
	_, err = activities.EnsureReclaimPolicyRetain(ctx, newPVC.Spec.VolumeName)
	if err != nil {
		panic(err)
	}

	// TODO: this is where rclone job goes

	// cache original pvc
	originalSpec, err := activities.GetPVC(ctx, "down-pvscope", "bogus")
	if err != nil {
		panic(err)
	}

	// drop both pvs
	err = activities.DeletePVC(ctx, "down-pvscope", "bogus")
	if err != nil {
		panic(err)
	}
	err = activities.DeletePVC(ctx, "down-pvscope", "bogus"+activities.StagingSuffix)
	if err != nil {
		panic(err)
	}
	time.Sleep(3 * time.Second) // TODO: again hack for wait to delete

	// map the new pv to the original pvc
	fmt.Println("Rebinding PVC")
	err = activities.RebindPV(ctx, "down-pvscope", newPVC.Spec.VolumeName, originalSpec, "0.5Gi")
	if err != nil {
		panic(err)
	}

	// TODO: optionally drop the original pv

	time.Sleep(10000 * time.Hour)
}
