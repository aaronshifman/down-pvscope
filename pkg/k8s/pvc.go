package k8s

import (
	"context"
	"fmt"
	"time"

	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

// createPVCandWait creates a pvc in k8s and waits until the PV is bound before returning
// if either the pvc fails to create or the pvc fails to bind "fast enough" (2mins) this errors
func CreatePVCandWait(ctx context.Context, client kubernetes.Interface, ns string, pvc *corev1.PersistentVolumeClaim) error {
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

// DeletePVCandWait drops a pvc in k8s and waits until the PVC no longer exists
// if either the pvc fails to delete or the pvc fails to delete "fast enough" (2mins) this errors
func DeletePVCandWait(ctx context.Context, client kubernetes.Interface, ns string, pvc string) error {
	err := client.CoreV1().PersistentVolumeClaims(ns).Delete(ctx, pvc, metav1.DeleteOptions{})
	if k8errors.IsNotFound(err) {
		fmt.Println("PVC already deleted")
		return nil
	} else if err != nil {
		return errors.Wrap(err, "Unable to drop pvc")
	}

	err = wait.PollUntilContextTimeout(ctx, 1*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		_, err := client.CoreV1().PersistentVolumeClaims(ns).Get(ctx, pvc, metav1.GetOptions{})

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
