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
	newPVC, err := activities.CreateStagingPVC(ctx, *originalPVC, "0.5Gi")
	if err != nil {
		panic(err)
	}
	fmt.Println(newPVC.Spec.VolumeName)

	// make sure new PV is safe
	// TODO: set the original reclaim policy on this PV
	_, err = activities.EnsureReclaimPolicyRetain(ctx, newPVC.Spec.VolumeName)
	if err != nil {
		panic(err)
	}

	// TODO: this is where rclone job goes

	// drop both pvs
	err = activities.DeletePVC(ctx, "down-pvscope", originalPVC.Name)
	if err != nil {
		panic(err)
	}
	err = activities.DeletePVC(ctx, "down-pvscope", newPVC.Name)
	if err != nil {
		panic(err)
	}

	// map the new pv to the original pvc
	fmt.Println("Rebinding PVC")
	err = activities.RebindPV(ctx, "down-pvscope", newPVC.Spec.VolumeName, originalPVC, "0.5Gi")
	if err != nil {
		panic(err)
	}

	// TODO: optionally drop the original pv

	time.Sleep(10000 * time.Hour)
}
