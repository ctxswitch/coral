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
	"errors"
	"time"

	imageClient "ctx.sh/coral/pkg/agent/client"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/limiter"
	"ctx.sh/coral/pkg/util"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const (
	DefaultRequeueAfter = time.Second * 5
)

type Options struct {
	Limiter                  *limiter.Limiter
	ImageClient              imageClient.ImageClient
	MaxConcurrentPullers     int
	MaxConcurrentReconcilers int
	NodeName                 string
}

type Request struct {
	types.NamespacedName
}

type Watcher struct {
	processor   *limiter.Limiter
	nodeName    string
	imageClient imageClient.ImageClient
	client.Client
}

func SetupWithManager(mgr ctrl.Manager, opts *Options) error {
	w := &Watcher{
		processor:   opts.Limiter,
		nodeName:    opts.NodeName,
		imageClient: opts.ImageClient,
		Client:      mgr.GetClient(),
	}

	h := handler.TypedFuncs[*coralv1beta1.ImageSync, Request]{
		CreateFunc: func(ctx context.Context, e event.TypedCreateEvent[*coralv1beta1.ImageSync], w workqueue.TypedRateLimitingInterface[Request]) {
			// Do nothing.  We handle when the finalizer is added in the update.
		},
		UpdateFunc: func(ctx context.Context, e event.TypedUpdateEvent[*coralv1beta1.ImageSync], w workqueue.TypedRateLimitingInterface[Request]) {
			w.Add(Request{
				NamespacedName: types.NamespacedName{
					Name:      e.ObjectNew.GetName(),
					Namespace: e.ObjectNew.GetNamespace(),
				},
			})
		},
		DeleteFunc: func(ctx context.Context, e event.TypedDeleteEvent[*coralv1beta1.ImageSync], w workqueue.TypedRateLimitingInterface[Request]) {
			w.Add(Request{
				NamespacedName: types.NamespacedName{
					Name:      e.Object.GetName(),
					Namespace: e.Object.GetNamespace(),
				},
			})
		},
		GenericFunc: func(ctx context.Context, e event.TypedGenericEvent[*coralv1beta1.ImageSync], w workqueue.TypedRateLimitingInterface[Request]) {
			// Do nothing
		},
	}

	return builder.TypedControllerManagedBy[Request](mgr).
		WatchesRawSource(source.TypedKind(
			mgr.GetCache(),
			&coralv1beta1.ImageSync{},
			h),
		).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Named("imagesync-watcher").
		Complete(w)
}

// +kubebuilder:rbac:groups=coral.ctx.sh,resources=imagesyncs,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (w *Watcher) Reconcile(ctx context.Context, req Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx, "name", req.Name, "namespace", req.Namespace)

	observed := NewObservedState()
	observer := StateObserver{
		Client:   w.Client,
		NodeName: w.nodeName,
		Request:  req,
	}

	err := observer.observe(ctx, observed)
	if err != nil {
		switch {
		case errors.Is(err, ErrNodeMatch):
			// I'm treating this as an error just for now.  Previously I was using this to clean up
			// the imagesync if the selectors were changed, but not anymore.  Still, I want to keep
			// this here for now.
			return ctrl.Result{}, nil
		case errors.Is(err, ErrImageSyncNotFound):
			// Imagesync has been deleted. Just return and do nothing.
			return ctrl.Result{}, nil
		case errors.Is(err, ErrPullSecretsNotFound):
			// Pull secrets have been specified but none of them were found.  Return error.
			log.Error(err, "pull secrets not found")
			return ctrl.Result{}, err
		default:
			log.Error(err, "unable to observe state", "request", req)
			return ctrl.Result{}, err
		}
	}

	// Handle the images that are being deleted.
	if !observed.ImageSync.DeletionTimestamp.IsZero() {
		log.V(2).Info("imagesync is being deleted, cleaning up")
		return ctrl.Result{}, nil
	}

	return w.process(ctx, observed.ImageSync, observed.PullSecrets)
}

func (w *Watcher) process(ctx context.Context, obj *coralv1beta1.ImageSync, pullSecrets []corev1.Secret) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	images, err := w.imageClient.List(ctx)
	if err != nil {
		log.Error(err, "failed to list images")
		return ctrl.Result{}, err
	}

	available := make(map[string]bool)
	for _, img := range images {
		available[img] = true
	}

	auth, err := NewAuth(pullSecrets)
	if err != nil {
		return ctrl.Result{}, err
	}

	eg, ctx := errgroup.WithContext(ctx)

	for _, img := range obj.Spec.Images {
		fqn := util.GetImageQualifiedName(util.DefaultSearchRegistry, img)
		if !available[fqn] {
			eg.Go(func() error {
				// TODO: Maybe pull this out so we don't create the routine if we can't acquire a processing slot.
				w.processor.Acquire()
				defer w.processor.Release()

				return w.addImage(ctx, fqn, auth)
			})
		}
	}

	if err := eg.Wait(); err != nil {
		log.Error(err, "failed to process imagesync")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (w *Watcher) addImage(ctx context.Context, fqn string, auth *Auth) error {
	log := ctrl.LoggerFrom(ctx, "name", fqn)
	log.V(2).Info("adding image")

	creds := auth.Lookup(fqn)
	if err := w.imageClient.Pull(ctx, fqn, creds); err != nil {
		return err
	}

	return nil
}
