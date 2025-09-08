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

package watcher

import (
	"context"

	"ctx.sh/coral/pkg/agent/client"
	"ctx.sh/coral/pkg/agent/watcher/imagesync"
	"ctx.sh/coral/pkg/limiter"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Options struct {
	ContainerAddr            string
	MaxConcurrentReconcilers int
	MaxConcurrentPullers     int
	NodeName                 string
}

type Watcher struct{}

func SetupWithManager(ctx context.Context, mgr ctrl.Manager, opts *Options) error {
	imageClient := client.New()
	if err := imageClient.Connect(ctx, opts.ContainerAddr); err != nil {
		return err
	}

	el := limiter.New(opts.MaxConcurrentPullers)

	if err := imagesync.SetupWithManager(mgr, &imagesync.Options{
		Limiter:                  el,
		MaxConcurrentReconcilers: opts.MaxConcurrentReconcilers,
		ImageClient:              imageClient,
		NodeName:                 opts.NodeName,
	}); err != nil {
		return err
	}

	return nil
}
