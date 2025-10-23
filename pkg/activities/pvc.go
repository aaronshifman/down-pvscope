package activities

import (
	"context"

	"github.com/aaronshifman/down-pvscope/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GetMatchingPV(ctx context.Context, pvcName, ns string) (string, error) {
	client, err := util.GetClientset()
	if err != nil {
		return "", err
	}

	pvc, err := client.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return pvc.Spec.VolumeName, nil
}

func EnsureReclaimPolicyRetain(ctx context.Context, pvName string) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	pv, err := client.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	if pv.Spec.PersistentVolumeReclaimPolicy != corev1.PersistentVolumeReclaimRetain {
		pv.Spec.PersistentVolumeReclaimPolicy = corev1.PersistentVolumeReclaimRetain
		_, err = client.CoreV1().PersistentVolumes().Update(ctx, pv, metav1.UpdateOptions{})
	}
	return err
}
