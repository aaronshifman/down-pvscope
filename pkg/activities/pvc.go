package activities

import (
	"context"
	"log/slog"

	"github.com/aaronshifman/down-pvscope/pkg/k8s"
	"github.com/aaronshifman/down-pvscope/pkg/util"
	"github.com/pkg/errors"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVCActivities struct{}

const stagingSuffix = "-staging"

func (a *PVCActivities) CreateStagingPVC(ctx context.Context, originalPVC util.PvcInfo, size string) (*util.PvcInfo, error) {
	client, err := util.GetClientset()
	if err != nil {
		return nil, err
	}

	// clone the pvc but change name + set volume size
	originalPVC.VolumeName = ""
	originalPVC.Name = originalPVC.Name + stagingSuffix
	originalPVC.Annotations = nil
	originalPVC.RequestedStorage = size
	originalPVC.LimitStorage = size
	slog.InfoContext(ctx, "Creating staging PVC", "name", originalPVC.Name, "newSize", size)

	pvc, err := originalPVC.ToK8s()
	if err != nil {
		return nil, errors.Wrap(err, "Unable to convert metadata to true k8s resource")
	}

	err = k8s.CreatePVCandWait(ctx, client, originalPVC.Namespace, pvc)
	if err != nil && !k8errors.IsAlreadyExists(err) {
		return nil, err
	}

	// reget the new pvc so we get an updated volume name
	newPVC, err := client.CoreV1().PersistentVolumeClaims(originalPVC.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "Unable to update the pvc reference")
	}

	return util.NewPVCInfo(newPVC), nil
}

func (a *PVCActivities) DeletePVC(ctx context.Context, namespace, pvcName string) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}
	return k8s.DeletePVCandWait(ctx, client, namespace, pvcName)
}

func (a *PVCActivities) RebindPV(ctx context.Context, namespace, pvName string, origPVC util.PvcInfo, newSize string) error {
	slog.DebugContext(ctx, "Binding original PVC name to new PV", "pv", pvName, "pvc", origPVC.Name)
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	slog.DebugContext(ctx, "Unlinking original PV", "pv", origPVC.VolumeName)
	if err := k8s.UnlinkPV(ctx, client, origPVC.VolumeName); err != nil {
		return err
	}
	slog.DebugContext(ctx, "Unlinking new PV", "pv", pvName)
	if err := k8s.UnlinkPV(ctx, client, pvName); err != nil {
		return err
	}

	slog.InfoContext(ctx, "Creating new PVC to match original", "name", origPVC.Name, "pv", pvName)
	origPVC.VolumeName = pvName
	origPVC.Annotations = nil
	origPVC.RequestedStorage = newSize
	origPVC.LimitStorage = newSize

	pvc, err := origPVC.ToK8s()
	if err != nil {
		return errors.Wrap(err, "Unable to convert metadata to k8s resource")
	}
	return k8s.CreatePVCandWait(ctx, client, namespace, pvc)
}

func (a *PVCActivities) GetPVC(ctx context.Context, namespace, pvcName string) (*util.PvcInfo, error) {
	client, err := util.GetClientset()
	if err != nil {
		return nil, err
	}

	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return util.NewPVCInfo(pvc), err
}
