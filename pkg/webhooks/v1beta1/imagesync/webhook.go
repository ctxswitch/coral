package imagesync

import (
	"context"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"fmt"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

type Webhook struct{}

func (w *Webhook) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&coralv1beta1.ImageSync{}).
		WithValidator(w).
		WithDefaulter(w).
		Complete()
}

func (w *Webhook) Default(ctx context.Context, obj runtime.Object) error {
	imageSync, ok := obj.(*coralv1beta1.ImageSync)
	if !ok {
		return fmt.Errorf("expected *coralv1beta1.ImageSync, got %v", obj)
	}

	coralv1beta1.Defaulted(imageSync)
	return nil
}

// ValidateCreate implements webhook Validator.
func (w *Webhook) ValidateCreate(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	warnings := make(admission.Warnings, 0)
	return warnings, nil
}

// ValidateUpdate implements webhook Validator.
func (w *Webhook) ValidateUpdate(ctx context.Context, old runtime.Object, new runtime.Object) (admission.Warnings, error) {
	warnings := make(admission.Warnings, 0)
	return warnings, nil
}

// ValidateDelete implements webhook Validator.
func (w *Webhook) ValidateDelete(ctx context.Context, obj runtime.Object) (admission.Warnings, error) {
	return nil, nil
}

var _ admission.CustomDefaulter = &Webhook{}
var _ webhook.CustomValidator = &Webhook{}
