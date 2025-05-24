package imagesync

import (
	"context"
	"ctx.sh/coral/pkg/agent/image"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/queue"
	"ctx.sh/coral/pkg/util"
	"errors"
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
	"strings"
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

type Request struct {
	types.NamespacedName
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

	h := handler.TypedFuncs[*coralv1beta1.ImageSync, Request]{
		CreateFunc: func(ctx context.Context, e event.TypedCreateEvent[*coralv1beta1.ImageSync], w workqueue.TypedRateLimitingInterface[Request]) {
			// Do nothing
		},
		UpdateFunc: func(ctx context.Context, e event.TypedUpdateEvent[*coralv1beta1.ImageSync], w workqueue.TypedRateLimitingInterface[Request]) {
			obj := e.ObjectNew
			if len(obj.Status.Images) > 0 {
				w.Add(Request{
					NamespacedName: types.NamespacedName{
						Name:      obj.GetName(),
						Namespace: obj.GetNamespace(),
					},
				})
			}
		},
		DeleteFunc: func(ctx context.Context, e event.TypedDeleteEvent[*coralv1beta1.ImageSync], w workqueue.TypedRateLimitingInterface[Request]) {
			obj := e.Object
			w.Add(Request{
				NamespacedName: types.NamespacedName{
					Name:      obj.GetName(),
					Namespace: obj.GetNamespace(),
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
		if errors.Is(err, ErrNodeMatch) {
			// Node doesn't match, ensure that we've cleaned up any labels in case the
			// selectors were changed.
			if w.collection.HasUID(string(observed.ImageSync.GetUID())) {
				return ctrl.Result{}, w.delete(ctx, observed.ImageSync)
			}

			return ctrl.Result{}, nil
		} else if errors.Is(err, ErrImageSyncNotFound) {
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

// We could get images from the collection and update all the labels...
// 1. Filter out all of the imagesync labels.
// 2. Go over all the collection, get the images.
// 3. Compare them with images on the node and add the images
func (w *Watcher) updateLabels(ctx context.Context, obj *coralv1beta1.ImageSync) error {
	node := new(corev1.Node)
	if err := w.Get(ctx, client.ObjectKey{Name: w.nodeName}, node); err != nil {
		return err
	}

	node = node.DeepCopy()

	// Remove the imagesync labels.
	labels := node.GetLabels()
	for k, _ := range labels {
		if strings.Contains(k, coralv1beta1.ImageSyncLabel) {
			delete(labels, k)
		}
	}

	// Get all of the unique images in the reference collection and add them.
	for _, img := range w.collection.ToImageList() {
		fqn := util.GetImageQualifiedName(util.DefaultSearchRegistry, img)
		ref := util.GetImageLabelValue(fqn)
		labels[coralv1beta1.ImageSyncLabel+"/"+ref] = "present"
	}

	node.SetLabels(labels)
	return w.Update(ctx, node)
}
