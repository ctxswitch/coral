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

	"github.com/prometheus/client_golang/prometheus"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

type Options struct {
	Registry string
}

type Controller struct {
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Registry string
	crclient.Client
}

func SetupWithManager(mgr ctrl.Manager, opts *Options) error {
	c := &Controller{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("mirror-controller"),
		Registry: opts.Registry,
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&coralv1beta1.Mirror{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(c)
}

// +kubebuilder:rbac:groups=coral.ctx.sh,resources=mirrors,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=coral.ctx.sh,resources=mirrors/status,verbs=get;update;patch

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	// Loop through the images in the Mirror spec and ensure that they are mirrored to coral.
	// TODO(rob): Decide how to handle any removals.
	logger := ctrl.LoggerFrom(ctx)
	logger.V(4).Info("reconciling mirror", "request", req)

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

	mirror := observed.Mirror.DeepCopy()

	// The image has been deleted.
	if observed.Mirror == nil {
		return ctrl.Result{}, nil
	}

	if !controllerutil.ContainsFinalizer(mirror, coralv1beta1.MirrorFinalizer) {
		return ctrl.Result{Requeue: true}, c.addFinalizer(ctx, mirror)
	}

	if !mirror.DeletionTimestamp.IsZero() {
		return ctrl.Result{}, c.removeFinalizer(ctx, mirror)
	}

	syncer := NewSynchronizer().
		WithDestinationRegistry(c.Registry).
		WithCopyAll(*observed.Mirror.Spec.CopyAllArchitectures).
		WithImagePullSecrets(observed.Secrets)

	syncError := false
	for _, image := range observed.Mirror.Spec.Images {
		if err := syncer.Copy(ctx, image); err != nil {
			logger.Error(err, "failed to sync image", "image", image)
			syncError = true
		}
	}

	if syncError {
		return ctrl.Result{RequeueAfter: 10 * time.Second}, nil
	}

	return ctrl.Result{}, nil
}

func (c *Controller) addFinalizer(ctx context.Context, mirror *coralv1beta1.Mirror) error {
	controllerutil.AddFinalizer(mirror, coralv1beta1.MirrorFinalizer)
	if err := c.Update(ctx, mirror); err != nil {
		return err
	}

	return nil
}

func (c *Controller) removeFinalizer(ctx context.Context, mirror *coralv1beta1.Mirror) error {
	if controllerutil.ContainsFinalizer(mirror, coralv1beta1.MirrorFinalizer) {
		if err := c.finalize(ctx, mirror); err != nil {
			return err
		}

		controllerutil.RemoveFinalizer(mirror, coralv1beta1.MirrorFinalizer)
		if err := c.Update(ctx, mirror); err != nil {
			return err
		}
	}

	return nil
}

func (c *Controller) finalize(ctx context.Context, mirror *coralv1beta1.Mirror) error {
	// TODO: I'm not currently using the finalizers.  This is in place in case we want to
	//   clean up the registry after the deletion.
	return nil
}
