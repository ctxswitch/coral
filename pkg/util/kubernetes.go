// Copyright 2025 Coral Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
