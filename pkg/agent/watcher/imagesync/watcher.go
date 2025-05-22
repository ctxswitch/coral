package imagesync

import (
	"context"
	"ctx.sh/coral/pkg/agent/image"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/queue"
	"golang.org/x/sync/errgroup"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/credentialprovider"
	"k8s.io/kubernetes/pkg/credentialprovider/secrets"
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
	authMap     map[types.NamespacedName][]*runtime.AuthConfig
	// TODO: this is now a global keyring, do we need to scope this to the
	//   imagesync resource?
	keyring credentialprovider.DockerKeyring

	client.Client
}

func SetupWithManager(mgr ctrl.Manager, opts *Options) error {
	w := &Watcher{
		workQueue:   opts.WorkQueue,
		nodeName:    opts.NodeName,
		imageClient: opts.ImageClient,
		authMap:     make(map[types.NamespacedName][]*runtime.AuthConfig),

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

// +kubebuilder:rbac:groups=coral.ctx.sh,resources=imagesyncs,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch

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
		log.V(4).Info("node selector does not match")
		if err := w.delete(ctx, isync); err != nil {
			log.Error(err, "failed to delete imagesync")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	if !isync.DeletionTimestamp.IsZero() {
		log.V(2).Info("imagesync is being deleted, cleaning up")
		if err := w.delete(ctx, isync); err != nil {
			log.Error(err, "failed to delete imagesync")
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	} else {
		pullSecrets, err := getPullSecrets(ctx, w.Client, isync)
		if err != nil {
			log.Error(err, "failed to get pull secrets")
			return ctrl.Result{}, err
		}

		defaultKeyring := credentialprovider.NewDockerKeyring()

		keyring, err := secrets.MakeDockerKeyring(pullSecrets, defaultKeyring)
		if err != nil {
			return ctrl.Result{}, err
		}

		log.V(2).Info("processing imagesync")
		if err := w.add(ctx, isync, keyring); err != nil {
			log.Error(err, "failed to delete imagesync")
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func (w *Watcher) add(ctx context.Context, obj *coralv1beta1.ImageSync, keyring credentialprovider.DockerKeyring) error {
	eg, ctx := errgroup.WithContext(ctx)

	uid := string(obj.GetUID())

	for _, img := range obj.Status.Images {
		eg.Go(func() error {
			w.workQueue.Acquire()
			defer w.workQueue.Release()

			auth := w.runtimeAuthLookup(img.Image, keyring)
			return w.imageClient.Pull(ctx, uid, img.Image, img.Reference, auth)
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

// TODO: I think I need to pull this up to the watcher.  We don't access k8s resources here.
func (w *Watcher) authLookup(name string, keyring credentialprovider.DockerKeyring) []credentialprovider.AuthConfig {
	auth, found := keyring.Lookup(name)
	if !found {
		return []credentialprovider.AuthConfig{}
	}

	return auth
}

func (w *Watcher) runtimeAuthLookup(name string, keyring credentialprovider.DockerKeyring) []*runtime.AuthConfig {
	// TODO: should probably cache this, but for now, it's not super expensive.
	auth := w.authLookup(name, keyring)
	runtimeAuth := make([]*runtime.AuthConfig, len(auth))
	for i, v := range auth {
		runtimeAuth[i] = &runtime.AuthConfig{
			Username:      v.Username,
			Password:      v.Password,
			Auth:          v.Auth,
			ServerAddress: v.ServerAddress,
			IdentityToken: v.IdentityToken,
			RegistryToken: v.RegistryToken,
		}
	}

	return runtimeAuth
}

func getPullSecrets(ctx context.Context, c client.Client, isync *coralv1beta1.ImageSync) ([]corev1.Secret, error) {
	all := make([]corev1.Secret, 0)

	pullSecrets := isync.Spec.ImagePullSecrets

	if pullSecrets == nil {
		return []corev1.Secret{}, nil
	}

	for _, s := range pullSecrets {
		secret := &corev1.Secret{}
		err := c.Get(ctx, client.ObjectKey{Name: s.Name, Namespace: isync.GetNamespace()}, secret)
		if err != nil {
			return []corev1.Secret{}, err
		}

		all = append(all, *secret.DeepCopy())
	}

	return all, nil
}
