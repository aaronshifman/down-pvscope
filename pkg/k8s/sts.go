package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

func ScaleSTS(ctx context.Context, client kubernetes.Interface, ns, name string, replicas int32) error {
	stsClient := client.AppsV1().StatefulSets(ns)
	sts, err := stsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to get StatefulSet")
	}

	sts.Spec.Replicas = &replicas
	_, err = stsClient.Update(ctx, sts, metav1.UpdateOptions{})
	if err != nil {
		return errors.Wrap(err, "failed to scale down StatefulSet")
	}

	// Wait for it to be fully scaled
	err = wait.PollUntilContextTimeout(ctx, 5*time.Second, 5*time.Minute, true, func(ctx context.Context) (bool, error) {
		current, err := stsClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		if current.Status.Replicas == replicas && current.Status.ReadyReplicas == replicas {
			return true, nil
		}
		return false, nil
	})
	if err != nil {
		return errors.Wrap(err, "timed out waiting for StatefulSet to scale")
	}

	return nil
}

func GetReplicas(ctx context.Context, client kubernetes.Interface, ns, name string) (int32, error) {
	stsClient := client.AppsV1().StatefulSets(ns)

	sts, err := stsClient.Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return 0, fmt.Errorf("failed to get StatefulSet %q: %w", name, err)
	}

	originalReplicas := int32(0)
	if sts.Spec.Replicas != nil {
		originalReplicas = *sts.Spec.Replicas
	}

	slog.DebugContext(ctx, "Found replicas", "count", originalReplicas)
	return originalReplicas, nil
}
