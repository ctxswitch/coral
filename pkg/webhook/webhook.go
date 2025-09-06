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

package webhook

import (
	"context"
	"ctx.sh/coral/pkg/webhook/v1beta1/registry"
	"fmt"

	"ctx.sh/coral/pkg/store"
	coral "ctx.sh/coral/pkg/webhook/v1beta1/coral"
	"ctx.sh/coral/pkg/webhook/v1beta1/imagesync"
	"ctx.sh/coral/pkg/webhook/v1beta1/injector"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/manager"
)

type Options struct {
	RegistryPort int
	NodeRef      *store.NodeRef
}

// +kubebuilder:webhook:verbs=create;update,path=/mutate-coral-ctx-sh-v1beta1-imagesync,mutating=true,failurePolicy=fail,matchPolicy=Equivalent,groups=coral.ctx.sh,resources=imagesyncs,versions=v1,name=mimagesync.coral.ctx.sh,admissionReviewVersions=v1beta1,sideEffects=none
// +kubebuilder:webhook:verbs=create;update,path=/validate-coral-ctx-sh-v1beta1-imagesync,mutating=false,failurePolicy=fail,matchPolicy=Equivalent,groups=coral.ctx.sh,resources=imagesyncs,versions=v1,name=vimagesync.coral.ctx.sh,admissionReviewVersions=v1beta1,sideEffects=none

// TODO: implement the injector for volume mounts.
// webhook:verbs=create;update,path=/inject-coral-ctx-sh-v1beta1-imagesync,mutating=true,failurePolicy=ignore,matchPolicy=Equivalent,groups=apps;batch,resources=cronjobs;daemonsets;deployments;jobs;replicasets;replicationcontrollers;statefulsets,versions=v1,name=minjector.coral.ctx.sh,admissionReviewVersions=v1beta1,sideEffects=none

func SetupWebhooksWithManager(ctx context.Context, mgr manager.Manager, opts *Options) error {
	// Setup check endpoints for health and readiness
	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		return fmt.Errorf("could not set up health check: %v", err)
	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		return fmt.Errorf("could not set up readiness check: %v", err)
	}

	// Setup admission webhooks
	w := imagesync.Webhook{}
	err := w.SetupWebhookWithManager(mgr)
	if err != nil {
		return err
	}

	if err := injector.SetupWebhookWithManager(mgr); err != nil {
		return fmt.Errorf("could not set up injector webhook: %v", err)
	}

	// Register coral services with the webhook server
	if err := coral.SetupWebhookWithManager(mgr, &coral.Options{
		NodeRef: opts.NodeRef,
	}); err != nil {
		return fmt.Errorf("could not set up coral services: %v", err)
	}

	// Register the registry service
	if err := registry.SetupWebhookWithManager(ctx, mgr, &registry.Options{
		Port: opts.RegistryPort,
	}); err != nil {
		return fmt.Errorf("could not set up registry webhook: %v", err)
	}

	return nil
}
