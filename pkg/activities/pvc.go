package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/aaronshifman/down-pvscope/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
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

	if err = createPVCandWait(ctx, client, originalPVC.Namespace, pvc); err != nil {
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

	err = client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
	if err != nil {
		return errors.Wrap(err, "Unable to drop pvc")
	}

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvcName, metav1.GetOptions{})

		// resource hasn't been deleted yet
		if err == nil {
			return false, nil
		}

		// resource no longer on cluster
		if k8errors.IsNotFound(err) {
			return true, nil
		}

		return false, err
	})
	return err
}

func RebindPV(ctx context.Context, namespace, pvName string, origPVC *corev1.PersistentVolumeClaim, newSize string) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	if err := unlinkPV(ctx, client, pvName); err != nil {
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

	return createPVCandWait(ctx, client, namespace, newPVC)
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

// createPVCandWait creates a pvc in k8s and waits until the PV is bound before returning
// if either the pvc fails to create or the pvc fails to bind "fast enough" (2mins) this errors
func createPVCandWait(ctx context.Context, client *kubernetes.Clientset, ns string, pvc *corev1.PersistentVolumeClaim) error {
	_, err := client.CoreV1().PersistentVolumeClaims(ns).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "Could not create pvc")
	}

	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		vc, err := client.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvc.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if vc.Status.Phase == corev1.ClaimBound {
			return true, nil
		}

		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "PVC never entered bound state")
	}
	return nil
}
