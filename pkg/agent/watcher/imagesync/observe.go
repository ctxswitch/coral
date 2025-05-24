package imagesync

import (
	"context"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type ObservedState struct {
	ImageSync   *coralv1beta1.ImageSync
	PullSecrets []corev1.Secret
	Node        *corev1.Node
	ObserveTime time.Time
}

func NewObservedState() *ObservedState {
	return &ObservedState{
		ImageSync:   nil,
		PullSecrets: nil,
		Node:        nil,
		ObserveTime: time.Now(),
	}
}

type StateObserver struct {
	Client   client.Client
	NodeName string
	Request  ctrl.Request
}

func (o *StateObserver) observe(ctx context.Context, observed *ObservedState) error {
	observedNode, err := o.observeNode(ctx)
	if err != nil {
		return err
	}

	if !o.nodeIsReady(observedNode) {
		return ErrNodeNotReady
	}

	observed.Node = observedNode

	observedImageSync, err := o.observerImageSync(ctx)
	if err != nil {
		return err
	}

	if observedImageSync == nil {
		return nil
	}

	coralv1beta1.Defaulted(observedImageSync)
	observed.ImageSync = observedImageSync

	// TODO: not sure I like this here.
	matches, err := o.nodeMatches(observedNode, observedImageSync.Spec.NodeSelector)
	if err != nil {
		return err
	}
	if !matches {
		return ErrNodeMatch
	}

	observedPullSecrets, err := o.observePullSecrets(ctx, observed)
	if err != nil {
		return err
	}

	observed.PullSecrets = observedPullSecrets

	return nil
}

func (o *StateObserver) observerImageSync(ctx context.Context) (*coralv1beta1.ImageSync, error) {
	observedImageSync := new(coralv1beta1.ImageSync)
	err := o.Client.Get(ctx, o.Request.NamespacedName, observedImageSync)
	if err != nil {
		if errors.IsNotFound(err) {
			return nil, ErrImageSyncNotFound
		}
		return nil, err
	}

	return observedImageSync.DeepCopy(), nil
}

func (o *StateObserver) observePullSecrets(ctx context.Context, observed *ObservedState) ([]corev1.Secret, error) {
	log := ctrl.LoggerFrom(ctx)

	observedPullSecrets := make([]corev1.Secret, 0)

	for _, pullSecret := range observed.ImageSync.Spec.ImagePullSecrets {
		secret := new(corev1.Secret)
		err := o.Client.Get(ctx, client.ObjectKey{
			Name:      pullSecret.Name,
			Namespace: observed.ImageSync.Namespace,
		}, secret)
		if err != nil {
			if client.IgnoreNotFound(err) != nil {
				return nil, err
			}
			log.V(3).Info("pull secret not found", "name", pullSecret.Name, "namespace", observed.ImageSync.Namespace)
			continue
		}

		observedPullSecrets = append(observedPullSecrets, *secret.DeepCopy())
	}

	return observedPullSecrets, nil

}

func (o *StateObserver) observeNode(ctx context.Context) (*corev1.Node, error) {
	node := new(corev1.Node)
	err := o.Client.Get(ctx, client.ObjectKey{Name: o.NodeName}, node)
	if err != nil {
		if errors.IsNotFound(err) {
			return node, ErrNodeNotFound
		}
		return node, err
	}

	return node.DeepCopy(), nil
}

func (o *StateObserver) nodeIsReady(node *corev1.Node) bool {
	conditionReady := false
	conditionNoDiskPressure := false
	conditionNoPIDPressure := false

	for _, condition := range node.Status.Conditions {
		switch condition.Type {
		case corev1.NodeReady:
			conditionReady = condition.Status == corev1.ConditionTrue
		case corev1.NodeDiskPressure:
			conditionNoDiskPressure = condition.Status == corev1.ConditionFalse
		case corev1.NodePIDPressure:
			conditionNoPIDPressure = condition.Status == corev1.ConditionFalse
		}
	}

	return conditionReady && conditionNoDiskPressure && conditionNoPIDPressure
}

func (o *StateObserver) nodeMatches(node *corev1.Node, selectors []coralv1beta1.NodeSelector) (bool, error) {
	ls := labels.NewSelector()
	for _, v := range selectors {
		req, err := labels.NewRequirement(v.Key, v.Operator, v.Values)
		if err != nil {
			return false, err
		}
		ls = ls.Add(*req)
	}

	return ls.Matches(labels.Set(node.GetLabels())), nil
}
