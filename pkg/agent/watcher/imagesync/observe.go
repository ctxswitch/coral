package imagesync

import (
	"context"
	coralctxshv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"
)

type ObservedState struct {
	ImageSync   *coralctxshv1beta1.ImageSync
	PullSecrets []corev1.Secret
	ObserveTime time.Time
}

func NewObservedState() *ObservedState {
	return &ObservedState{
		ImageSync:   nil,
		PullSecrets: nil,
		ObserveTime: time.Now(),
	}
}

type StateObserver struct {
	Client  client.Client
	Request ctrl.Request
}

func (o *StateObserver) observe(ctx context.Context, observed *ObservedState) error {
	observedImageSync, err := o.observerImageSync(ctx)
	if err != nil {
		return err
	}

	if observedImageSync == nil {
		return nil
	}

	coralctxshv1beta1.Defaulted(observedImageSync)
	observed.ImageSync = observedImageSync

	observedPullSecrets, err := o.observePullSecrets(ctx, observed)
	if err != nil {
		return err
	}

	observed.PullSecrets = observedPullSecrets

	return nil
}

func (o *StateObserver) observerImageSync(ctx context.Context) (*coralctxshv1beta1.ImageSync, error) {
	observedImageSync := new(coralctxshv1beta1.ImageSync)
	err := o.Client.Get(ctx, o.Request.NamespacedName, observedImageSync)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return nil, err
		}
		return nil, nil
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
