package activities

import (
	"context"
	"log/slog"

	"github.com/aaronshifman/down-pvscope/pkg/k8s"
	"github.com/aaronshifman/down-pvscope/pkg/util"
)

type STSActivities struct{}

func (a *STSActivities) GetInitialReplicase(ctx context.Context, ns, sts string) (int32, error) {
	slog.DebugContext(ctx, "Getting replicas for", "name", sts, "namespace", ns)
	client, err := util.GetClientset()
	if err != nil {
		return 0, err
	}

	return k8s.GetReplicas(ctx, client, ns, sts)
}

func (a *STSActivities) ScaleTo0(ctx context.Context, ns, sts string) error {
	slog.DebugContext(ctx, "Getting replicas for", "name", sts, "namespace", ns)
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	return k8s.ScaleSTS(ctx, client, ns, sts, 0)
}

func (a *STSActivities) ScaleUp(ctx context.Context, ns, sts string, replicas int32) error {
	slog.DebugContext(ctx, "Scaling sts back up", "name", sts, "namespace", ns, "replicas", replicas)
	client, err := util.GetClientset()
	if err != nil {
		return err
	}

	return k8s.ScaleSTS(ctx, client, ns, sts, replicas)
}
