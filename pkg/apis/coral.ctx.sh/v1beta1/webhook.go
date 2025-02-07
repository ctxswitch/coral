// Copyright 2024 Coral Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package v1beta1

// +kubebuilder:docs-gen:collapse=Apache License

import (
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:docs-gen:collapse=Go imports

// +kubebuilder:webhook:verbs=create;update,path=/mutate-ctx-sh-v1-image,mutating=true,failurePolicy=fail,groups=coral.ctx.sh,resources=images,versions=v1beta1,name=mimage.coral.ctx.sh,admissionReviewVersions=v1beta1,sideEffects=none
// +kubebuilder:webhook:verbs=create;update,path=/validate-ctx-sh-v1-image,mutating=false,failurePolicy=fail,groups=coral.ctx.sh,resources=images,versions=v1beta1,name=vimage.coral.ctx.sh,admissionReviewVersions=v1beta1,sideEffects=none

// SetupWebhookWithManager adds webhook for BuildSet.
func (i *Image) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(i).
		Complete()
}

func (i *Image) Default() {
	Defaulted(i)
}

// ValidateCreate implements webhook Validator.
func (i *Image) ValidateCreate() (admission.Warnings, error) {
	warnings := make(admission.Warnings, 0)
	return warnings, nil
}

// ValidateUpdate implements webhook Validator.
func (i *Image) ValidateUpdate(old runtime.Object) (admission.Warnings, error) {
	warnings := make(admission.Warnings, 0)
	return warnings, nil
}

// ValidateDelete implements webhook Validator.
func (i *Image) ValidateDelete() (admission.Warnings, error) {
	return nil, nil
}
