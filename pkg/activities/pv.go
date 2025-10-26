package activities

import (
	"context"
	"fmt"
	"time"

	"github.com/aaronshifman/down-pvscope/pkg/util"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func EnsureReclaimPolicyRetain(ctx context.Context, pvName string) (corev1.PersistentVolumeReclaimPolicy, error) {
	client, err := util.GetClientset()
	if err != nil {
		return "", err
	}

	return setPVRetainPolicy(ctx, client, pvName, corev1.PersistentVolumeReclaimRetain)
}

func SetReclaimPolicy(ctx context.Context, pvName string, policy corev1.PersistentVolumeReclaimPolicy) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	_, err = setPVRetainPolicy(ctx, client, pvName, policy)
	return err
}

func unlinkPV(ctx context.Context, client *kubernetes.Clientset, pvName string) error {
	pv, err := client.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "Could not get PV")
	}

	fmt.Println("Unlinking existing volume")
	pv.Spec.ClaimRef = nil
	_, err = client.CoreV1().PersistentVolumes().Update(ctx, pv, metav1.UpdateOptions{})
	if err != nil {
		return err
	}

	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		v, err := client.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if v.Spec.ClaimRef == nil {
			return true, nil
		}

		return false, nil
	})
	return err
}

func setPVRetainPolicy(ctx context.Context, client *kubernetes.Clientset, pvName string, policy corev1.PersistentVolumeReclaimPolicy) (corev1.PersistentVolumeReclaimPolicy, error) {
	pv, err := client.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	originalPolicy := pv.Spec.PersistentVolumeReclaimPolicy
	if originalPolicy != policy {
		pv.Spec.PersistentVolumeReclaimPolicy = policy
		_, err = client.CoreV1().PersistentVolumes().Update(ctx, pv, metav1.UpdateOptions{})
		if err != nil {
			return "", errors.Wrap(err, "Unable to change PV retain policy")
		}
	} else {
		// early abort - the starting policy matches
		return policy, nil
	}

	err = wait.PollUntilContextTimeout(ctx, 2*time.Second, 2*time.Minute, true, func(ctx context.Context) (bool, error) {
		v, err := client.CoreV1().PersistentVolumes().Get(ctx, pvName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}

		if v.Spec.PersistentVolumeReclaimPolicy == policy {
			return true, nil
		}

		return false, nil
	})
	return originalPolicy, err
}
