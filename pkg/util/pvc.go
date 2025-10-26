package util

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PvcInfo struct {
	// Meta fields
	// NOTE: creating the PVC with existing annotations can cause issues
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`

	// Status fields
	VolumeName string `json:"volumeName"`

	// Spec fields
	StorageClassName string   `json:"storageClassName"`
	AccessModes      []string `json:"accessModes"`
	RequestedStorage string   `json:"requestedStorage"`
	LimitStorage     string   `json:"limitStorage"`
}

func NewPVCInfo(pvc *corev1.PersistentVolumeClaim) *PvcInfo {
	accessModes := []string{}
	for _, mode := range pvc.Spec.AccessModes {
		accessModes = append(accessModes, string(mode))
	}

	requestedStorage := ""
	if storage, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
		requestedStorage = storage.String()
	}

	limitStorage := ""
	if storage, ok := pvc.Spec.Resources.Limits[corev1.ResourceStorage]; ok {
		limitStorage = storage.String()
	}

	return &PvcInfo{
		Name:             pvc.Name,
		Namespace:        pvc.Namespace,
		Labels:           pvc.Labels,
		Annotations:      pvc.Annotations,
		VolumeName:       pvc.Spec.VolumeName,
		StorageClassName: *pvc.Spec.StorageClassName,
		AccessModes:      accessModes,
		RequestedStorage: requestedStorage,
		LimitStorage:     limitStorage,
	}
}

func (pvc *PvcInfo) ToK8s() (*corev1.PersistentVolumeClaim, error) {
	accessModes := make([]corev1.PersistentVolumeAccessMode, 0, len(pvc.AccessModes))
	for _, modeStr := range pvc.AccessModes {
		accessModes = append(accessModes, corev1.PersistentVolumeAccessMode(modeStr))
	}

	storageRequest, err := resource.ParseQuantity(pvc.RequestedStorage)
	if err != nil {
		return nil, err
	}

	storageLimit, err := resource.ParseQuantity(pvc.RequestedStorage)
	if err != nil {
		return nil, err
	}

	return &corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        pvc.Name,
			Namespace:   pvc.Namespace,
			Labels:      pvc.Labels,
			Annotations: pvc.Annotations,
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			AccessModes: accessModes,
			Resources: corev1.VolumeResourceRequirements{
				Limits: corev1.ResourceList{
					corev1.ResourceStorage: storageLimit,
				},
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: storageRequest,
				},
			},
			StorageClassName: &pvc.StorageClassName,
			VolumeName:       pvc.VolumeName,
		},
	}, nil
}
