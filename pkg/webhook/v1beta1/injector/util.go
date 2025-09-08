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

package injector

import (
	"errors"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ModifyFunc[S, T any](a []S, b []T, fn func(S, T) S) []S {
	var result []S
	for _, item := range a {
		for _, other := range b {
			result = append(result, fn(item, other))
		}
	}

	return result
}

func ObjectFromKind(kind string) (client.Object, error) {
	switch kind {
	case "CronJob":
		return &batchv1.CronJob{}, nil
	case "DaemonSet":
		return &appsv1.DaemonSet{}, nil
	case "Deployment":
		return &appsv1.Deployment{}, nil
	case "Job":
		return &batchv1.Job{}, nil
	case "ReplicaSet":
		return &appsv1.ReplicaSet{}, nil
	case "ReplicationController":
		return &corev1.ReplicationController{}, nil
	case "StatefulSet":
		return &appsv1.StatefulSet{}, nil
	case "Pod":
		return &corev1.Pod{}, nil
	default:
		return nil, errors.New("kind not supported")
	}
}
