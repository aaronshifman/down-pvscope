package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/aaronshifman/down-pvscope/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const StagingSuffix = "-staging"

func CreateStagingPVC(ctx context.Context, namespace, name, size string) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{Name: name + StagingSuffix},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: []corev1.PersistentVolumeAccessMode{corev1.ReadWriteOnce},
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{"storage": resource.MustParse(size)},
			},
		},
	}
	_, err = client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return errors.Wrap(err, "Could not create pvc")
	}

	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		vc, err := client.CoreV1().PersistentVolumeClaims(namespace).Get(ctx, pvc.Name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if vc.Status.Phase == corev1.ClaimBound {
			return true, nil
		}

		return false, nil
	})

	return err
}

func DeletePVC(ctx context.Context, namespace, pvcName string) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	return client.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvcName, metav1.DeleteOptions{})
}

func RebindPV(ctx context.Context, namespace, pvName string, origPVC *corev1.PersistentVolumeClaim, newSize string) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	pv, err := client.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// unlink existing volume
	fmt.Println("Unlinking existing volume")
	pv.Spec.ClaimRef = nil
	_, err = client.CoreV1().PersistentVolumes().Update(ctx, pv, metav1.UpdateOptions{})
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)

	// create new PVC
	fmt.Println("Creating cloned pvc")
	origPVC.Spec.VolumeName = pvName
	newPVC := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        origPVC.Name,
			Namespace:   origPVC.Namespace,
			Labels:      origPVC.Labels,
			Annotations: origPVC.Annotations,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes:      origPVC.Spec.AccessModes,
			StorageClassName: origPVC.Spec.StorageClassName,
			VolumeName:       pvName,
			Resources: corev1.VolumeResourceRequirements{
				Requests: corev1.ResourceList{"storage": resource.MustParse(newSize)},
			},
		},
	}

	_, err = client.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, newPVC, metav1.CreateOptions{})
	if err != nil {
		return err
	}

	return err
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
