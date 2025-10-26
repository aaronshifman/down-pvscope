package activities

import (
	"context"
	"fmt"

	"github.com/aaronshifman/down-pvscope/pkg/k8s"
	"github.com/aaronshifman/down-pvscope/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const StagingSuffix = "-staging"

func CreateStagingPVC(ctx context.Context, originalPVC corev1.PersistentVolumeClaim, size string) (corev1.PersistentVolumeClaim, error) {
	client, err := util.GetClientset()
	if err != nil {
		return corev1.PersistentVolumeClaim{}, err
	}

	// clone the pvc but change name + set volume size
	pvc := originalPVC.DeepCopy()
	pvc.Spec.VolumeName = ""
	pvc.ObjectMeta = metav1.ObjectMeta{Name: originalPVC.Name + StagingSuffix}
	pvc.Spec.Resources = corev1.VolumeResourceRequirements{
		Requests: corev1.ResourceList{"storage": resource.MustParse(size)},
		Limits:   corev1.ResourceList{"storage": resource.MustParse(size)},
	}

	if err = k8s.CreatePVCandWait(ctx, client, originalPVC.Namespace, pvc); err != nil {
		return corev1.PersistentVolumeClaim{}, err
	}

	// reget the new pvc so we get an updated volume name
	newPVC, err := client.CoreV1().PersistentVolumeClaims(originalPVC.Namespace).Get(ctx, pvc.Name, metav1.GetOptions{})
	if err != nil {
		return corev1.PersistentVolumeClaim{}, errors.Wrap(err, "Unable to update the pvc reference")
	}

	return *newPVC, nil
}

func DeletePVC(ctx context.Context, namespace, pvcName string) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}
	return k8s.DeletePVCandWait(ctx, client, namespace, pvcName)

}

func RebindPV(ctx context.Context, namespace, pvName string, origPVC *corev1.PersistentVolumeClaim, newSize string) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	if err := k8s.UnlinkPV(ctx, client, pvName); err != nil {
		return err
	}

	fmt.Println("Creating cloned pvc")
	newPVC := origPVC.DeepCopy()
	newPVC.Spec.VolumeName = pvName
	newPVC.ObjectMeta = metav1.ObjectMeta{
		Name:        origPVC.Name,
		Labels:      origPVC.Labels,
		Annotations: origPVC.Annotations,
	}
	newPVC.Spec.Resources = corev1.VolumeResourceRequirements{
		Requests: corev1.ResourceList{"storage": resource.MustParse(newSize)},
		Limits:   corev1.ResourceList{"storage": resource.MustParse(newSize)},
	}

	return k8s.CreatePVCandWait(ctx, client, namespace, newPVC)
}

func GetPVC(ctx context.Context, namespace, pvcName string) (*corev1.PersistentVolumeClaim, error) {
	client, err := util.GetClientset()
	if err != nil {
		return nil, err
	}

	pvc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return pvc, err
}
