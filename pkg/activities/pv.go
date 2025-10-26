package activities

import (
	"context"

	"github.com/aaronshifman/down-pvscope/pkg/k8s"
	"github.com/aaronshifman/down-pvscope/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

func EnsureReclaimPolicyRetain(ctx context.Context, pvName string) (corev1.PersistentVolumeReclaimPolicy, error) {
	client, err := util.GetClientset()
	if err != nil {
		return "", err
	}

	return k8s.SetPVRetainPolicy(ctx, client, pvName, corev1.PersistentVolumeReclaimRetain)
}

func SetReclaimPolicy(ctx context.Context, pvName string, policy corev1.PersistentVolumeReclaimPolicy) error {
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	_, err = k8s.SetPVRetainPolicy(ctx, client, pvName, policy)
	return err
}
