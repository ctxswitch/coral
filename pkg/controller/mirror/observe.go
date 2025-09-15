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

package mirror

import (
	"context"
	"time"

	coralctxshv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObservedState struct {
	Mirror      *coralctxshv1beta1.Mirror
	Secrets     []corev1.Secret
	ObserveTime time.Time
}

func NewObservedState() *ObservedState {
	return &ObservedState{
		Mirror:      nil,
		Secrets:     make([]corev1.Secret, 0),
		ObserveTime: time.Now(),
	}
}

type StateObserver struct {
	Client  client.Client
	Request ctrl.Request
}

func (o *StateObserver) observe(ctx context.Context, observed *ObservedState) error {
	observedMirror, err := o.getMirror(ctx)
	if err != nil {
		return err
	}

	// If mirror is not found, there's nothing to observe
	if observedMirror == nil {
		return nil
	}

	coralctxshv1beta1.Defaulted(observedMirror)
	observed.Mirror = observedMirror

	observedSecrets, err := o.getSecrets(ctx, observedMirror.Spec.ImagePullSecrets)
	if err != nil {
		return err
	}
	observed.Secrets = observedSecrets

	return nil
}

func (o *StateObserver) getMirror(ctx context.Context) (*coralctxshv1beta1.Mirror, error) {
	mirror := &coralctxshv1beta1.Mirror{}
	err := o.Client.Get(ctx, o.Request.NamespacedName, mirror)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}
		return nil, nil
	}

	return mirror, nil
}

func (o *StateObserver) getSecrets(ctx context.Context, imagePullSecrets []corev1.LocalObjectReference) ([]corev1.Secret, error) {
	secrets := make([]corev1.Secret, 0)
	for _, ref := range imagePullSecrets {
		if secret, err := o.getSecret(ctx, ref.Name); err != nil {
			if client.IgnoreNotFound(err) != nil {
				return nil, err
			}
		} else {
			secrets = append(secrets, *secret)
		}
	}

	return secrets, nil
}

func (o *StateObserver) getSecret(ctx context.Context, name string) (*corev1.Secret, error) {
	secret := corev1.Secret{}
	err := o.Client.Get(ctx, types.NamespacedName{
		Namespace: o.Request.Namespace,
		Name:      name,
	}, &secret)
	return &secret, err
}
