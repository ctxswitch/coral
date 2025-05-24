package imagesync

import (
	"context"
	"ctx.sh/coral/pkg/agent/image"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/queue"
	"errors"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
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
	"sync"
	"time"
)

const (
	DefaultRequeueAfter = time.Second * 5
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
	locker      sync.Mutex
	collection  *References
	client.Client
}

func SetupWithManager(mgr ctrl.Manager, opts *Options) error {
	w := &Watcher{
		workQueue:   opts.WorkQueue,
		nodeName:    opts.NodeName,
		imageClient: opts.ImageClient,
		collection:  NewReferences(),

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
		Client:   w.Client,
		NodeName: w.nodeName,
		Request:  req,
	}

	err := observer.observe(ctx, observed)
	if err != nil {
		// TODO: I still don't know if I like handling this as an error.
		if errors.Is(err, ErrNodeMatch) {
			// TODO: Cleanup handler. Ensure cleanup in case the node selectors have changed and we
			//   no longer match.  It's pretty brute force right now.  Need a reference map to check.
			return ctrl.Result{}, w.delete(ctx, observed.ImageSync)
		} else if errors.Is(err, ErrImageSyncNotFound) {
			// The resource has been deleted.
			// TODO: There's a chance that we haven't cleaned up labels so the node may actually
			//   get into a state where we have orphaned labels that may represent a stale state.
			return ctrl.Result{}, nil
		} else {
			log.Error(err, "unable to observe state", "request", req)
			return ctrl.Result{}, err
		}
	}

	// Handle the images that are being deleted.
	if !observed.ImageSync.DeletionTimestamp.IsZero() {
		log.V(2).Info("imagesync is being deleted, cleaning up")
		err := w.delete(ctx, observed.ImageSync)
		if err != nil {
			log.Error(err, "failed to delete imagesync")
			// TODO: update the labels regardless of the errors, but still handle the error.
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, w.updateLabels(ctx, observed.ImageSync)
	}

	auth, err := NewAuth(observed.PullSecrets)
	if err != nil {
		log.Error(err, "failed to create auth")
		return ctrl.Result{}, err
	}

	log.V(2).Info("processing imagesync")
	if err := w.add(ctx, observed.ImageSync, auth); err != nil {
		log.Error(err, "failed to add imagesync")
		// TODO: update the labels regardless of the errors, but still handle the error.
		return ctrl.Result{}, err
	}

	w.locker.Lock()
	defer w.locker.Unlock()

	return ctrl.Result{}, w.updateLabels(ctx, observed.ImageSync)
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
			info, err := w.imageClient.Pull(ctx, uid, img.Image, creds)
			if err != nil {
				return err
			}

			w.collection.Add(uid, info.Name, info.ID)
			return nil
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

			info, err := w.imageClient.Delete(ctx, uid, img.Image)
			if err != nil {
				return err
			}

			w.collection.Remove(uid, info.Name, info.ID)
			return err
		})
	}

	return eg.Wait()
}

func (w *Watcher) updateLabels(ctx context.Context, obj *coralv1beta1.ImageSync) error {
	node := new(corev1.Node)
	if err := w.Get(ctx, client.ObjectKey{Name: w.nodeName}, node); err != nil {
		return err
	}

	labels := node.GetLabels()

	node = node.DeepCopy()
	for _, img := range obj.Status.Images {
		// TODO: make sure this doesn't return error for not found.
		info, err := w.imageClient.Status(ctx, img.Image)
		if err != nil {
			return err
		}

		if w.collection.IsReferenced(info.Name, info.ID) {
			labels[coralv1beta1.ImageSyncLabel+"/"+img.Reference] = "present"
		} else {
			delete(labels, coralv1beta1.ImageSyncLabel+"/"+img.Reference)
		}
	}

	node.SetLabels(labels)
	return w.Update(ctx, node)
}
