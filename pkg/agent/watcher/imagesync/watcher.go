package imagesync

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"ctx.sh/coral/pkg/agent/image"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/queue"
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
		switch {
		case errors.Is(err, ErrNodeMatch):
			// Node doesn't match, ensure that we've cleaned up any labels in case the
			// selectors were changed.
			if w.collection.HasUID(string(observed.ImageSync.GetUID())) {
				return w.process(ctx, observed.ImageSync, observed.PullSecrets, true)
			}

			return ctrl.Result{}, nil
		case errors.Is(err, ErrImageSyncNotFound):
			return ctrl.Result{}, nil
		case errors.Is(err, ErrPullSecretsNotFound):
			log.Error(err, "pull secrets not found")
		default:
			log.Error(err, "unable to observe state", "request", req)
			return ctrl.Result{}, err
		}
	}

	// Handle the images that are being deleted.
	if !observed.ImageSync.DeletionTimestamp.IsZero() {
		log.V(2).Info("imagesync is being deleted, cleaning up")
		return w.process(ctx, observed.ImageSync, observed.PullSecrets, true)
	}

	return w.process(ctx, observed.ImageSync, observed.PullSecrets, false)
}

func (w *Watcher) process(ctx context.Context, obj *coralv1beta1.ImageSync, pullSecrets []corev1.Secret, deleted bool) (ctrl.Result, error) {
	uid := string(obj.GetUID())
	log := ctrl.LoggerFrom(ctx)

	seen := make(map[string]bool)
	for _, img := range w.collection.ImageListForUID(uid) {
		seen[img] = false
	}

	for _, img := range obj.Spec.Images {
		fqn := util.GetImageQualifiedName(util.DefaultSearchRegistry, img)
		seen[fqn] = true
	}

	log.V(4).Info("seen images", "map", seen, "deleted", deleted)

	auth, err := NewAuth(pullSecrets)
	if err != nil {
		return ctrl.Result{}, err
	}

	eg, ctx := errgroup.WithContext(ctx)

	for img, present := range seen {
		if !present || deleted {
			eg.Go(func() error {
				return w.deleteImage(ctx, uid, img)
			})
		} else {
			eg.Go(func() error {
				return w.addImage(ctx, uid, img, auth)
			})
		}
	}

	if err := eg.Wait(); err != nil {
		log.Error(err, "failed to process imagesync", "uid", uid)
		return ctrl.Result{}, err
	}

	w.locker.Lock()
	defer w.locker.Unlock()

	return ctrl.Result{}, w.updateLabels(ctx)
}

func (w *Watcher) addImage(ctx context.Context, uid, img string, auth *Auth) error {
	w.workQueue.Acquire()
	defer w.workQueue.Release()

	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("adding image", "image", img)

	creds := auth.Lookup(img)
	info, err := w.imageClient.Pull(ctx, uid, img, creds)
	if err != nil {
		return err
	}

	w.collection.Add(uid, info.Name, info.ID)
	return nil

}

func (w *Watcher) deleteImage(ctx context.Context, uid, img string) error {
	w.workQueue.Acquire()
	defer w.workQueue.Release()

	log := ctrl.LoggerFrom(ctx)
	log.V(2).Info("deleting image", "image", img)

	info, err := w.imageClient.Delete(ctx, uid, img)
	if err != nil {
		return err
	}

	w.collection.Remove(uid, info.Name, info.ID)
	return nil
}

func (w *Watcher) updateLabels(ctx context.Context) error {
	node := new(corev1.Node)
	if err := w.Get(ctx, client.ObjectKey{Name: w.nodeName}, node); err != nil {
		return err
	}

	node = node.DeepCopy()

	// Remove the imagesync labels.
	labels := node.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	for k := range labels {
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
