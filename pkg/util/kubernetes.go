package util

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func GroupKindToType(gk schema.GroupKind) (any, error) {
	switch gk {
	case batchv1.SchemeGroupVersion.WithKind("CronJob").GroupKind():
		return &batchv1.CronJob{}, nil
	case appsv1.SchemeGroupVersion.WithKind("DaemonSet").GroupKind():
		return &appsv1.DaemonSet{}, nil
	case appsv1.SchemeGroupVersion.WithKind("Deployment").GroupKind():
		return &appsv1.Deployment{}, nil
	case batchv1.SchemeGroupVersion.WithKind("Job").GroupKind():
		return &batchv1.Job{}, nil
	case appsv1.SchemeGroupVersion.WithKind("ReplicaSet").GroupKind():
		return &appsv1.ReplicaSet{}, nil
	case corev1.SchemeGroupVersion.WithKind("ReplicationController").GroupKind():
		return &corev1.ReplicationController{}, nil
	case appsv1.SchemeGroupVersion.WithKind("StatefulSet").GroupKind():
		return &appsv1.StatefulSet{}, nil
	case corev1.SchemeGroupVersion.WithKind("Pod").GroupKind():
		return &corev1.Pod{}, nil
	default:
		return nil, fmt.Errorf("unknown group kind %v", gk)
	}
}
