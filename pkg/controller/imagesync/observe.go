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

package imagesync

import (
	"context"
	"time"

	coralctxshv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type ObservedState struct {
	ImageSync   *coralctxshv1beta1.ImageSync
	ObserveTime time.Time
}

func NewObservedState() *ObservedState {
	return &ObservedState{
		ImageSync:   nil,
		ObserveTime: time.Now(),
	}
}

type StateObserver struct {
	Client  client.Client
	Request ctrl.Request
}

func (o *StateObserver) observe(ctx context.Context, observed *ObservedState) error {
	observedImageSync, err := o.observerImageSync(ctx)
	if err != nil {
		return err
	}

	if observedImageSync == nil {
		return nil
	}

	coralctxshv1beta1.Defaulted(observedImageSync)
	observed.ImageSync = observedImageSync

	return nil
}

func (o *StateObserver) observerImageSync(ctx context.Context) (*coralctxshv1beta1.ImageSync, error) {
	observedImageSync := new(coralctxshv1beta1.ImageSync)
	err := o.Client.Get(ctx, o.Request.NamespacedName, observedImageSync)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}
		return nil, nil
	}

	return observedImageSync.DeepCopy(), nil
}
