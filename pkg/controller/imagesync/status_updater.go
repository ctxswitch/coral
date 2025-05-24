package imagesync

import (
	"context"
	"time"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/client"

	// "sigs.k8s.io/controller-runtime/pkg/manager".
	"sync"
)

type StatusCheck map[types.NamespacedName]coralv1beta1.ImageSyncCondition

// StatusUpdater is a process that runs in the background to update the status of the
// imagesync on the required nodes.
type StatusUpdater struct {
	stopCh   chan struct{}
	stopOnce sync.Once
	client.Client
}

// NewStatusUpdater creates a new status updater.
func NewStatusUpdater(mgr ctrl.Manager) *StatusUpdater {
	return &StatusUpdater{
		Client: mgr.GetClient(),
		stopCh: make(chan struct{}),
	}
}

// Start starts the status updater process.
func (su *StatusUpdater) Start(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)
	log.V(4).Info("starting imagesync status updater")

	timer := time.NewTicker(5 * time.Second)

	for {
		select {
		case <-timer.C:
			if err := su.update(ctx); err != nil {
				log.Error(err, "failed to update status")
			}
		case <-ctx.Done():
			su.Stop()
		case <-su.stopCh:
			return nil
		}
	}
}

// Stop stops the status updater process.
func (su *StatusUpdater) Stop() {
	su.stopOnce.Do(func() {
		close(su.stopCh)
	})
}

func (su *StatusUpdater) update(ctx context.Context) error {
	// List all imagesyncs labeled as processed
	var imagesyncs coralv1beta1.ImageSyncList
	if err := su.List(ctx, &imagesyncs, client.MatchingLabels{
		coralv1beta1.ProcessedLabelName: coralv1beta1.ProcessedLabelValue,
	}); err != nil {
		return err
	}

	// List all the nodes.
	var nodes corev1.NodeList
	if err := su.List(ctx, &nodes); err != nil {
		return err
	}

	// TODO: Add async workers instead of a single loop for each run.  Handle this like
	//   we do in the agent (with the limiter).  Use error wait groups.

	for _, i := range imagesyncs.Items {
		filtered := su.filterNodes(nodes.Items, i.Spec.NodeSelector)

		images := make([]coralv1beta1.ImageSyncImage, 0)
		for _, image := range i.Status.Images {
			available := 0
			pending := 0

			// TODO: Revisit.  There's probably a better/more optimal way to do this.
			for _, node := range filtered {
				if _, ok := node.Labels[coralv1beta1.ImageSyncLabel+"/"+image.Reference]; ok {
					available++
				} else {
					pending++
				}
			}

			images = append(images, coralv1beta1.ImageSyncImage{
				Name:      image.Name,
				Image:     image.Image,
				Reference: image.Reference,
				Available: available,
				Pending:   pending,
			})
		}

		// Loop through the images and retrieve the number of nodes with all the images
		// available.
		available := len(filtered)
		for _, image := range images {
			available = min(available, image.Available)
		}

		status := i.Status.DeepCopy()
		status.TotalImages = len(i.Status.Images)
		status.TotalNodes = len(filtered)
		status.Condition.Available = available
		status.Condition.Pending = len(filtered) - available
		status.Images = images

		i.Status = *status

		// Update the status of the imagesync.
		if err := su.Client.Status().Update(ctx, &i); err != nil {
			// don't error, just log and continue.
			return err
		}
	}

	return nil
}

func (su *StatusUpdater) filterNodes(nodes []corev1.Node, selectors []coralv1beta1.NodeSelector) []corev1.Node {
	var filtered []corev1.Node
	for _, node := range nodes {
		if su.matches(node.Labels, selectors) {
			filtered = append(filtered, node)
		}
	}

	return filtered
}

func (su *StatusUpdater) matches(l map[string]string, selector []coralv1beta1.NodeSelector) bool {
	ls := labels.NewSelector()
	for _, v := range selector {
		req, err := labels.NewRequirement(v.Key, v.Operator, v.Values)
		if err != nil {
			return false
		}
		ls = ls.Add(*req)
	}

	return ls.Matches(labels.Set(l))
}
