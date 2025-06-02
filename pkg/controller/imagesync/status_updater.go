package imagesync

import (
	"context"
	"ctx.sh/coral/pkg/store"
	"ctx.sh/coral/pkg/util"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"reflect"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/manager"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
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
	nodeRef  *store.NodeRef
	stopCh   chan struct{}
	stopOnce sync.Once
	client.Client
}

// NewStatusUpdater creates a new status updater.
func NewStatusUpdater(c client.Client, ref *store.NodeRef) *StatusUpdater {
	return &StatusUpdater{
		Client:  c,
		nodeRef: ref,
		stopCh:  make(chan struct{}),
	}
}

// NeedLeaderElection returns true to indicate that this runnable should run
// when the controller manager is the leader.
func (su *StatusUpdater) NeedLeaderElection() bool {
	return true
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
	// TODO: Wait on init lock.

	log := ctrl.LoggerFrom(ctx)

	var isyncs coralv1beta1.ImageSyncList
	if err := su.Client.List(ctx, &isyncs); err != nil {
		return err
	}

	if len(isyncs.Items) == 0 {
		return nil
	}

	var nodes corev1.NodeList
	if err := su.Client.List(ctx, &nodes); err != nil {
		return err
	}

	for _, item := range isyncs.Items {
		isync := item.DeepCopy()
		log := log.WithValues("name", isync.GetName(), "namespace", isync.GetNamespace())

		filteredNodes := su.filterNodes(nodes.Items, isync.Spec.NodeSelector)

		status := coralv1beta1.ImageSyncStatus{
			TotalNodes:  len(filteredNodes),
			TotalImages: len(isync.Spec.Images),
			Condition: coralv1beta1.ImageSyncCondition{
				Available: 0,
				Pending:   0,
			},
		}

		if len(filteredNodes) == 0 {
			ctrl.LoggerFrom(ctx).V(4).Info("no nodes match the imagesync node selector")
			return su.updateStatus(ctx, isync, status)
		}

		available := make(map[string]int)
		for _, img := range isync.Spec.Images {
			fqn := util.GetImageQualifiedName(util.DefaultSearchRegistry, img)
			available[fqn] = 0
		}

		images := make([]coralv1beta1.ImageSyncImage, 0)
		for _, image := range isync.Spec.Images {
			fqn := util.GetImageQualifiedName(util.DefaultSearchRegistry, image)
			for _, node := range filteredNodes {
				if su.nodeRef.HasImage(node.Name, fqn) {
					available[fqn]++
				}
			}

			images = append(images, coralv1beta1.ImageSyncImage{
				Image:     fqn,
				Available: available[fqn],
				Pending:   len(filteredNodes) - available[fqn],
			})
		}

		// Get the number of nodes with all images available.
		minNodes := len(filteredNodes)
		for _, count := range available {
			minNodes = min(minNodes, count)
		}

		status.Images = images
		status.Condition = coralv1beta1.ImageSyncCondition{
			Available: minNodes,
			Pending:   len(filteredNodes) - minNodes,
		}

		log.V(5).Info("updating imagesync status", "status", status)
		if err := su.updateStatus(ctx, isync, status); err != nil {
			ctrl.LoggerFrom(ctx).Error(err, "failed to update imagesync status")
		}
	}

	return nil
}

func (su *StatusUpdater) updateStatus(ctx context.Context, isync *coralv1beta1.ImageSync, status coralv1beta1.ImageSyncStatus) error {
	if !reflect.DeepEqual(isync.Status, status) {
		status.DeepCopyInto(&isync.Status)
		isync.Status.LastUpdated = metav1.Now()
		return su.Client.Status().Update(ctx, isync)
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

var _ manager.Runnable = &StatusUpdater{}
var _ manager.LeaderElectionRunnable = &StatusUpdater{}
