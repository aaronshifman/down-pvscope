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
	pv, err := activities.GetMatchingPV(ctx, "bogus", "down-pvscope")
	if err != nil {
		panic(err)
	}
	fmt.Println(pv)

	// mark existing safe
	// TODO: store original status and restore at end
	err = activities.EnsureReclaimPolicyRetain(ctx, pv)
	if err != nil {
		panic(err)
	}

	// create new PVC / provision new PV
	err = activities.CreateStagingPVC(ctx, "down-pvscope", "bogus", "0.5Gi")
	if err != nil {
		panic(err)
	}

	// find new pv
	time.Sleep(3 * time.Second) // TODO: hack wait for pvc to provision
	newPV, err := activities.GetMatchingPV(ctx, "bogus"+activities.StagingSuffix, "down-pvscope")
	if err != nil {
		panic(err)
	}
	fmt.Println(newPV)

	// make sure new PV is safe
	err = activities.EnsureReclaimPolicyRetain(ctx, newPV)
	if err != nil {
		panic(err)
	}

	// TODO: this is where rclone job goes

	// cache original pvc
	originalSpec, err := activities.GetPVC(ctx, "down-pvscope", "bogus")

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
	err = activities.RebindPV(ctx, "down-pvscope", newPV, originalSpec, "0.5Gi")
	if err != nil {
		panic(err)
	}

	// TODO: optionally drop the original pv

	time.Sleep(10000 * time.Hour)
}
