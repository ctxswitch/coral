package image

import (
	"context"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	stvziov1 "stvz.io/coral/pkg/apis/stvz.io/v1"
	"stvz.io/coral/pkg/monitor"
	"stvz.io/coral/pkg/util"
)

const (
	Finalizer = "image.stvz.io/finalizer"
)

type Controller struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Monitor  *monitor.Manager
}

func SetupWithManager(mgr ctrl.Manager, mtr *monitor.Manager) error {
	c := &Controller{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("image-controller"),
		Monitor:  mtr,
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&stvziov1.Image{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(c)
}

// +kubebuilder:rbac:groups=stvz.io,resources=images,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=stvz.io,resources=images/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=stvz.io,resources=images/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get

// Reconcile is the main controller loop for the image controller.
func (c Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(6).Info("reconciling image", "request", req)

	observed := NewObservedState()
	observer := StateObserver{
		Client:  c.Client,
		Request: req,
	}

	err := observer.observe(ctx, observed)
	if err != nil {
		logger.Error(err, "unable to observe state", "request", req)
		return ctrl.Result{
			RequeueAfter: 10 * time.Second,
		}, err
	}

	logger.V(8).Info("observed image", "obj", observed.image)

	// TODO: I think there'a an issue here.  I'm using this to catch any potential races
	// where the image has been deleted and can't be found, but I'm not sure this is the
	// only case where a nil value could happen without an error.  I think that's the case
	// but not quite sure.
	if observed.image == nil {
		logger.V(8).Info("image not found, exiting reconcile loop")
		return ctrl.Result{}, nil
	}

	// TODO: Because we don't do anything with the image we could just return without
	// a requeue here.  Check this out later.
	if observed.image.DeletionTimestamp.IsZero() {
		has := controllerutil.ContainsFinalizer(observed.image, Finalizer)
		if !has {
			logger.V(8).Info("adding finalizer and monitor", "finalizer", Finalizer)
			controllerutil.AddFinalizer(observed.image, Finalizer)
			err := c.Client.Update(ctx, observed.image)
			if err != nil {
				// TODO: to requue or not?
				return ctrl.Result{
					RequeueAfter: 10 * time.Second,
				}, err
			}
		}
	} else {
		// The image is being deleted, wait for the nodes to clean up the images
		// before removing the finalizer.
		if controllerutil.ContainsFinalizer(observed.image, Finalizer) {
			logger.V(8).Info("waiting for nodes to remove the images, shutting down monitor, and removing finalizer", "finalizer", Finalizer)
			err := c.finish(ctx, observed.image)
			if err != nil && err.Error() == ErrNodesNotEmpty.Error() {
				logger.V(6).Info("nodes still have images, waiting for cleanup")
				return ctrl.Result{
					RequeueAfter: 10 * time.Second,
				}, nil
			} else if err != nil {
				return ctrl.Result{
					RequeueAfter: 10 * time.Second,
				}, err
			}
		}
		return ctrl.Result{}, nil
	}

	// Place the image resource under monitoring
	c.Monitor.AddImage(ctx, observed.image)

	return ctrl.Result{}, nil
}

func (c *Controller) finish(ctx context.Context, image *stvziov1.Image) error {
	logger := log.FromContext(ctx)

	// I only care if there are nodes that still have any images present?
	selectors := labels.NewSelector()

	// If the image has selectors, we need to add those.
	if image.Spec.Selector != nil {
		for _, selector := range image.Spec.Selector {
			req, err := labels.NewRequirement(selector.Key, selector.Operator, selector.Values)
			if err != nil {
				return err
			}
			selectors = selectors.Add(*req)
		}
	}

	// TODO: There's a condition here where if the image is also assigned to a node
	// by another object, then the image would not be deleted and we would be stuck
	// here forever.  We could potentially get around this by adding a name/namespace
	// itentifier to the label?  Will revisit this later.
	for _, i := range image.Spec.Images {
		for _, tag := range i.Tags {
			tagSelectors := selectors.DeepCopySelector()
			label := util.HashedImageLabelKey(*i.Name + ":" + tag)
			reqs, err := labels.NewRequirement(label, selection.Exists, nil)
			if err != nil {
				return err
			}
			tagSelectors = tagSelectors.Add(*reqs)

			// If there are nodes that still have the image present, then we don't delete
			// the finalizer.  This will keep the image resource around so the node worker
			// that has not yet removed the images can use the information contained in the
			// resource to do so.  This does have the side effect of potentially not deleting
			// the image resource if the node is in a bad state or the worker is unable to
			// clean.  This is a tradeoff that we are willing to make in this case as it
			// allows us to visualize that the entire cluster is in a consistent state.  I could
			// set a timeout to remove the finalizer if the nodes are not cleaned up in a certain
			// amount of time by using the deletion timestamp.
			nodes := new(corev1.NodeList)
			err = c.Client.List(ctx, nodes, &client.ListOptions{LabelSelector: tagSelectors})
			if err != nil {
				return err
			}

			if len(nodes.Items) > 0 {
				return ErrNodesNotEmpty
			}
		}
	}

	logger.V(8).Info("removing monitor and finalizer")
	c.Monitor.RemoveImage(image)
	controllerutil.RemoveFinalizer(image, Finalizer)
	return c.Client.Update(ctx, image)
}
