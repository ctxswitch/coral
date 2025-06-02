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

	"ctx.sh/coral/pkg/store"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Options struct {
	NodeRef *store.NodeRef
}

type Controller struct {
	Cache    cache.Cache
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	client.Client
}

func SetupWithManager(mgr ctrl.Manager, opts *Options) error {
	if err := mgr.Add(NewStatusUpdater(mgr.GetClient(), opts.NodeRef)); err != nil {
		return err
	}

	c := &Controller{
		Cache:    mgr.GetCache(),
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("imagesync-controller"),
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&coralv1beta1.ImageSync{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(c)
}

// +kubebuilder:rbac:groups=coral.ctx.sh,resources=imagesyncs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coral.ctx.sh,resources=imagesyncs/status,verbs=get;update;patch

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(4).Info("reconciling image", "request", req)

	observed := NewObservedState()
	observer := StateObserver{
		Client:  c.Client,
		Request: req,
	}

	err := observer.observe(ctx, observed)
	if err != nil {
		logger.Error(err, "unable to observe state", "request", req)
		observerError.With(prometheus.Labels{
			"name":      req.Name,
			"namespace": req.Namespace,
		}).Inc()
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}

	isync := observed.ImageSync

	// The image has been deleted.
	if observed.ImageSync == nil {
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(isync, coralv1beta1.ImageSyncFinalizer) {
		return ctrl.Result{Requeue: true}, c.addFinalizer(ctx, isync)
	}

	if !isync.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, c.removeFinalizer(ctx, isync)
	}

	return ctrl.Result{}, nil
}

func (c *Controller) addFinalizer(ctx context.Context, isync *coralv1beta1.ImageSync) error {
	controllerutil.AddFinalizer(isync, coralv1beta1.ImageSyncFinalizer)
	if err := c.Update(ctx, isync); err != nil {
		return err
	}

	return nil
}

func (c *Controller) removeFinalizer(ctx context.Context, isync *coralv1beta1.ImageSync) error {
	if controllerutil.ContainsFinalizer(isync, coralv1beta1.ImageSyncFinalizer) {
		if err := c.finalize(ctx, isync); err != nil {
			return err
		}

		controllerutil.RemoveFinalizer(isync, coralv1beta1.ImageSyncFinalizer)
		if err := c.Update(ctx, isync); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) finalize(ctx context.Context, isync *coralv1beta1.ImageSync) error {
	// TODO: I'm not currently using the finalizers, but it does trigger an update event
	//   with the deletion time set that the agents can catch for cleanup.
	return nil
}
