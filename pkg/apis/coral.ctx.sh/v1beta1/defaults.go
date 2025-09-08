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
	"k8s.io/utils/ptr"
)

// +kubebuilder:docs-gen:collapse=Go imports

func defaultedImageSyncSpec(obj *ImageSyncSpec) {}

func defaultedImageSync(obj *ImageSync) {
	defaultedImageSyncSpec(&obj.Spec)
}

func defaultedMirrorSpec(obj *MirrorSpec) {
	if obj.CopyAllArchitectures == nil {
		obj.CopyAllArchitectures = ptr.To(false)
	}
}

func defaultedMirror(obj *Mirror) {
	defaultedMirrorSpec(&obj.Spec)
}

// Defaulted sets the resource defaults.
func Defaulted(obj runtime.Object) {
	switch obj := obj.(type) { //nolint:gocritic
	case *ImageSync:
		defaultedImageSync(obj)
	case *Mirror:
		defaultedMirror(obj)
	}
}
