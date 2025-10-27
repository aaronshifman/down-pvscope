package activities

import (
	"context"
	"log/slog"

	"github.com/aaronshifman/down-pvscope/pkg/k8s"
	"github.com/aaronshifman/down-pvscope/pkg/util"
	corev1 "k8s.io/api/core/v1"
)

type PVActivities struct{}

func (pva *PVActivities) EnsureReclaimPolicyRetain(ctx context.Context, pvName string) (corev1.PersistentVolumeReclaimPolicy, error) {
	slog.DebugContext(ctx, "Marking pv retain", "name", pvName)
	client, err := util.GetClientset()
	if err != nil {
		return "", err
	}

	return k8s.SetPVRetainPolicy(ctx, client, pvName, corev1.PersistentVolumeReclaimRetain)
}

func (pva *PVActivities) SetReclaimPolicy(ctx context.Context, pvName string, policy corev1.PersistentVolumeReclaimPolicy) error {
	slog.DebugContext(ctx, "Marking pv to policy", "name", pvName, "policy", policy)
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	_, err = k8s.SetPVRetainPolicy(ctx, client, pvName, policy)
	return err
}
