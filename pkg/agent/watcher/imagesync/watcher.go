package imagesync

import (
	"context"
	"ctx.sh/coral/pkg/agent/image"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/queue"
	"golang.org/x/sync/errgroup"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
	"time"
)

type Options struct {
	WorkQueue                *queue.Queue
	ImageClient              image.ImageClient
	MaxConcurrentPullers     int
	MaxConcurrentReconcilers int
	NodeName                 string
}

type Watcher struct {
	workQueue   *queue.Queue
	nodeName    string
	imageClient image.ImageClient
	client.Client
}

func SetupWithManager(mgr ctrl.Manager, opts *Options) error {
	w := &Watcher{
		workQueue:   opts.WorkQueue,
		nodeName:    opts.NodeName,
		imageClient: opts.ImageClient,

		Client: mgr.GetClient(),
	}

	src := source.Kind(
		mgr.GetCache(),
		&coralv1beta1.ImageSync{},
		handler.TypedFuncs[*coralv1beta1.ImageSync, reconcile.Request]{
			UpdateFunc: func(ctx context.Context, e event.TypedUpdateEvent[*coralv1beta1.ImageSync], q workqueue.TypedRateLimitingInterface[reconcile.Request]) {
				log := ctrl.LoggerFrom(ctx)
				if e.ObjectNew.IsProcessed() {
					log.V(4).Info("update added request to queue", "object", e.ObjectNew)
					q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
						Name:      e.ObjectNew.Name,
						Namespace: e.ObjectNew.Namespace,
					}})
				}
			},
			CreateFunc:  nil,
			DeleteFunc:  nil,
			GenericFunc: nil,
		},
	)

	return ctrl.NewControllerManagedBy(mgr).
		For(&coralv1beta1.ImageSync{}).
		WatchesRawSource(src).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		WithOptions(controller.Options{
			MaxConcurrentReconciles: opts.MaxConcurrentReconcilers,
		}).
		Complete(w)
}

// +kubebuilder:rbac:groups=coral.ctx.sh,resources=imagesyncs,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

func (w *Watcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	observed := NewObservedState()
	observer := StateObserver{
		Client:  w.Client,
		Request: req,
	}

	err := observer.observe(ctx, observed)
	if err != nil {
		log.Error(err, "unable to observe state", "request", req)
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}

	if observed.ImageSync == nil {
		return ctrl.Result{}, nil
	}

	ok, err := w.imageClient.Matches(ctx, observed.ImageSync.Spec.NodeSelector)
	if err != nil {
		log.Error(err, "failed to match node selector")
		return ctrl.Result{}, err
	}

	if !ok {
		// Ensure that we don't have any images for this imagesync resource in case
		// the node selector has changed.  Probably a better way to track this, but
		// brute force works for now.
		log.V(4).Info("node selector does not match")
		if err := w.delete(ctx, observed.ImageSync); err != nil {
			log.Error(err, "failed to delete imagesync")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if !observed.ImageSync.DeletionTimestamp.IsZero() {
		log.V(2).Info("imagesync is being deleted, cleaning up")
		err := w.delete(ctx, observed.ImageSync)
		if err != nil {
			log.Error(err, "failed to delete imagesync")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	auth, err := NewAuth(observed.PullSecrets)
	if err != nil {
		log.Error(err, "failed to create auth")
		return ctrl.Result{}, err
	}

	log.V(2).Info("processing imagesync")
	if err := w.add(ctx, observed.ImageSync, auth); err != nil {
		log.Error(err, "failed to delete imagesync")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (w *Watcher) add(ctx context.Context, obj *coralv1beta1.ImageSync, auth *Auth) error {
	eg, ctx := errgroup.WithContext(ctx)

	uid := string(obj.GetUID())
	log := ctrl.LoggerFrom(ctx)

	for _, img := range obj.Status.Images {
		eg.Go(func() error {
			w.workQueue.Acquire()
			defer w.workQueue.Release()

			creds := auth.Lookup(img.Image)
			log.V(4).Info("worker is processing image")
			return w.imageClient.Pull(ctx, uid, img.Image, img.Reference, creds)
		})
	}

	return eg.Wait()
}

func (w *Watcher) delete(ctx context.Context, obj *coralv1beta1.ImageSync) error {
	eg, ctx := errgroup.WithContext(ctx)

	uid := string(obj.GetUID())

	for _, img := range obj.Status.Images {
		eg.Go(func() error {
			w.workQueue.Acquire()
			defer w.workQueue.Release()

			return w.imageClient.Delete(ctx, uid, img.Image, img.Reference)
		})
	}

	return eg.Wait()
}

// func (w *Watcher) filter(ctx context.Context, obj *coralv1beta1.ImageSync) []ctrl.Request {
// 	// Only return objects that have been processed through the core reconciler. Technically
// 	// we would only need this if relying on status information that is added by the controller.
// 	// Even though we are not right now, I'll keep this in just in case we want something in
// 	// the future.
// 	log := ctrl.LoggerFrom(ctx)
// 	if obj.IsProcessed() {
// 		log.Info("filter matched")
// 		return []ctrl.Request{{
// 			NamespacedName: types.NamespacedName{
// 				Name:      obj.GetName(),
// 				Namespace: obj.GetNamespace(),
// 			},
// 		}}
// 	}
// 	log.V(4).Info("filter did not match, empty request", "object", obj)
// 	return nil
// }
//
// func (w *Watcher) predicate(ctx context.Context) predicate.Predicate {
// 	log := ctrl.LoggerFrom(ctx)
//
// 	return predicate.Funcs{
// 		CreateFunc: func(e event.CreateEvent) bool {
// 			return false
// 		},
// 		UpdateFunc: func(e event.UpdateEvent) bool {
// 			// Only process updates if the status has changed.
// 			if e.ObjectOld.GetResourceVersion() == e.ObjectNew.GetResourceVersion() {
// 				log.V(4).Info("update predicate, no change")
// 				return false
// 			}
// 			log.V(4).Info("update predicate")
// 			return true
// 		},
// 		DeleteFunc: func(e event.DeleteEvent) bool {
// 			// Only process deletes if the object is not being deleted.
// 			if e.Object.GetDeletionTimestamp() != nil {
// 				log.V(4).Info("delete predicate, delete timestamp not nil")
// 				return false
// 			}
// 			log.V(4).Info("delete predicate")
// 			return true
// 		},
// 		GenericFunc: func(e event.GenericEvent) bool {
// 			return false
// 		},
// 	}
// }
