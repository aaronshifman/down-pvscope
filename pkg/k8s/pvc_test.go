package k8s_test

import (
	"context"
	"testing"

	"github.com/aaronshifman/down-pvscope/pkg/k8s"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestCreatePVC(t *testing.T) {
	client := fake.NewSimpleClientset()
	pvc := &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-pvc",
			Namespace: "foo",
		},
		Spec: corev1.PersistentVolumeClaimSpec{},
	}

	testCases := []struct {
		Name string
		Fake func()
		Ok   bool
	}{
		{
			Name: "ok",
			Fake: func() {
				client.ClearActions()
				client.ReactionChain = []k8stesting.Reactor{}
				client.PrependReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					obj := pvc.DeepCopy()
					obj.Status.Phase = corev1.ClaimBound
					obj.Name = obj.Name + "-staging"
					return true, obj, nil
				})
			},
			Ok: true,
		},
		{
			Name: "failedtocreate",
			Fake: func() {
				client.ClearActions()
				client.ReactionChain = []k8stesting.Reactor{}
				client.PrependReactor("get", "persistentvolumeclaims", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
					return true, nil, errors.New("somethings's broken")
				})
			},
			Ok: false,
		},
	}

	for _, tt := range testCases {
		t.Run(tt.Name, func(t *testing.T) {
			tt.Fake()
			err := k8s.CreatePVCandWait(context.Background(), client, "foo", pvc)
			if tt.Ok {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}

