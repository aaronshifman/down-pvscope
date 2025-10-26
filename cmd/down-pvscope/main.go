package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/aaronshifman/down-pvscope/pkg/activities"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "namespace",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "pvc",
				Required: true,
			},
			&cli.StringFlag{
				Name:     "size",
				Required: true,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			namespace := cmd.String("namespace")
			pvc := cmd.String("pvc")
			size := cmd.String("size")

			workflow(ctx, namespace, pvc, size)
			return nil
		},
	}
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

// pretend workflow until this is hooked up to temporal
func workflow(ctx context.Context, namespace, pvc, size string) {
	// find existing
	originalPVC, err := activities.GetPVC(ctx, namespace, pvc)
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
	newPVC, err := activities.CreateStagingPVC(ctx, *originalPVC, size)
	if err != nil {
		panic(err)
	}
	fmt.Println(newPVC.Spec.VolumeName)

	// make sure new PV is safe
	// TODO: set the original reclaim policy on this PV
	originalPolicy, err := activities.EnsureReclaimPolicyRetain(ctx, newPVC.Spec.VolumeName)
	if err != nil {
		panic(err)
	}

	fmt.Println("Starting rclone job")
	err = activities.Runrclone(ctx, originalPVC, &newPVC, namespace)
	if err != nil {
		panic(err)
	}

	// drop both pvs
	err = activities.DeletePVC(ctx, namespace, originalPVC.Name)
	if err != nil {
		panic(err)
	}
	err = activities.DeletePVC(ctx, namespace, newPVC.Name)
	if err != nil {
		panic(err)
	}

	// map the new pv to the original pvc
	fmt.Println("Rebinding PVC")
	err = activities.RebindPV(ctx, namespace, newPVC.Spec.VolumeName, originalPVC, size)
	if err != nil {
		panic(err)
	}

	fmt.Println("Resetting the PV retain policy")
	err = activities.SetReclaimPolicy(ctx, newPVC.Spec.VolumeName, originalPolicy)
	if err != nil {
		panic(err)
	}

	// TODO: optionally drop the original pv

	fmt.Println("Done: sleeping")

	time.Sleep(10000 * time.Hour)
}
