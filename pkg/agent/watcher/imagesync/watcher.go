package imagesync

import (
	"context"
	"sync"

	"ctx.sh/coral/pkg/agent/image"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/queue"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/source"
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
		handler.TypedEnqueueRequestsFromMapFunc(w.filter),
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

// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch

func (w *Watcher) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)

	isync := &coralv1beta1.ImageSync{}
	if err := w.Get(ctx, req.NamespacedName, isync); err != nil {
		if client.IgnoreNotFound(err) == nil {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	ok, err := w.imageClient.Matches(ctx, isync.Spec.NodeSelector)
	if err != nil {
		log.Error(err, "failed to match node selector")
		return ctrl.Result{}, err
	}

	if !ok {
		// Ensure that we don't have any images for this imagesync resource in case
		// the node selector has changed.
		log.V(4).Info("node selector does not match, ensuring we no longer are managing images")

		w.delete(ctx, isync)
		return ctrl.Result{}, nil
	}

	// TODO: is it possible that I miss a delete event?
	if !isync.DeletionTimestamp.IsZero() {
		log.V(2).Info("imagesync is being deleted, cleaning up")
		w.delete(ctx, isync)
		return ctrl.Result{}, nil
	} else {
		log.V(2).Info("processing imagesync")
		w.add(ctx, isync)
	}

	return ctrl.Result{}, nil
}

func (w *Watcher) add(ctx context.Context, obj *coralv1beta1.ImageSync) {
	wg := sync.WaitGroup{}

	for _, img := range obj.Spec.Images {
		// TODO: Is there a possibility that we wind up with orphans by adding here?  We need fqn as key?
		w.workQueue.Acquire()
		wg.Add(1)
		go func() {
			defer w.workQueue.Release()
			defer wg.Done()
			// TODO: If there are errors requeue.
			_ = w.imageClient.Pull(ctx, string(obj.GetUID()), img)
		}()
	}
	wg.Wait()
}

func (w *Watcher) delete(ctx context.Context, obj *coralv1beta1.ImageSync) {
	wg := sync.WaitGroup{}
	for _, img := range obj.Spec.Images {
		// TODO: This can go away and we will handle the reference deletion here.  If there
		//   are no more references, then we mark it with a tombstone.
		w.workQueue.Acquire()
		wg.Add(1)
		go func() {
			defer w.workQueue.Release()
			defer wg.Done()
			// TODO: If there are errors requeue.
			_ = w.imageClient.Delete(ctx, string(obj.GetUID()), img)
		}()
	}
	wg.Wait()
}

func (w *Watcher) filter(ctx context.Context, obj *coralv1beta1.ImageSync) []ctrl.Request {
	// Only return objects that have been processed through the core reconciler. Technically
	// we would only need this if relying on status information that is added by the controller.
	// Even though we are not right now, I'll keep this in just in case we want something in
	// the future.
	if obj.IsProcessed() {
		return []ctrl.Request{{
			NamespacedName: types.NamespacedName{
				Name:      obj.GetName(),
				Namespace: obj.GetNamespace(),
			},
		}}
	}
	return []ctrl.Request{}
}

// func (w *Watcher) isManaged(ctx context.Context, obj *coralv1beta1.ImageSync) bool {
// 	return false
// }
