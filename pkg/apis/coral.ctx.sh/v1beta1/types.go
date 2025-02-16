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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"
)

// +kubebuilder:docs-gen:collapse=Go imports

const (
	ImageSyncFinalizer = "imagesync.coral.ctx.sh/finalizer"
	MirrorFinalizer    = "mirror.coral.ctx.sh/finalizer"
)

type NodeSelector struct {
	Key      string             `json:"key"`
	Operator selection.Operator `json:"operator"`
	Values   []string           `json:"values"`
}

type ListSelector string

const (
	ListSelectorAll ListSelector = "all"
)

// ImageSyncSpec is the spec for a Image resource.
type ImageSyncSpec struct {
	// +required
	// +listType=atomic
	// Images to fetch.
	Images []string `json:"images"`
	// +optional
	// +nullable
	// Selector defines which nodes the image should be synced to.
	Selector []NodeSelector `json:"selector"`
	// +optional
	// +nullable
	// ImagePullSecrets is a list of secrets to use when pulling the image.
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true
// +kubebuilder:validation:Required
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=img,singular=images
// +kubebuilder:printcolumn:name="Images",type="integer",JSONPath=".status.totalImages",description="The number of total images managed by the object"
// +kubebuilder:printcolumn:name="Available",type="integer",JSONPath=".status.condition.available",description="The number of images that are currently available on the nodes"
// +kubebuilder:printcolumn:name="Pending",type="integer",JSONPath=".status.condition.pending",description="The number of images that are currently pending on the nodes"
// +kubebuilder:printcolumn:name="Unknown",type="integer",JSONPath=".status.condition.unknown",description="The number of images that are in an unknown state on the nodes",priority=1
// +kubebuilder:printcolumn:name="Nodes",type="integer",JSONPath=".status.totalNodes",description="The number of nodes matching the selector (if any)",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// ImageSync is an external image that will be mirrored to each configured node.
type ImageSync struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ImageSyncSpec `json:"spec"`
	// +optional
	Status ImageSyncStatus `json:"status"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type ImageSyncList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ImageSync `json:"items"`
}

type ImageSyncState string

const (
	ImageStatePending   ImageSyncState = "pending"
	ImageStateAvailable ImageSyncState = "available"
	ImageStateUnknown   ImageSyncState = "unknown"
)

// ImageSyncCondition represents the condition of the sync.
type ImageSyncCondition struct {
	// +required
	// Available is the number of images that are currently available on the nodes.
	Available int `json:"available"`
	// +required
	// Pending is the number of images that are currently pending on the nodes.
	Pending int `json:"pending"`
	// +required
	// Unknown is the number of images that are in an unknown state on the nodes.
	Unknown int `json:"unknown"`
}

// ImageSyncStatus is the status for a WatchSet resource.
type ImageSyncStatus struct {
	// +optional
	// TotalNodes is the number of nodes that should have the image prefetched.
	TotalNodes int `json:"totalNodes"`
	// +optional
	// TotalImages is the number of total images managed by the object.
	TotalImages int `json:"totalImages"`
	// +optional
	// Condition is the current state of the images on the nodes.
	Condition ImageSyncCondition `json:"condition"`
	// +optional
	// Data is a list of image data that will be used to track the images on the nodes.
	// Data []ImageSyncData `json:"data"`
}

type MirrorSpec struct {
	// +required
	// LocalRegistry is the url to the local registry
	LocalRegistry string `json:"localRegistry"`
	// +optional
	// +nullable
	// ImagePullSecrets is a list of secrets to use when pulling the image.
	// ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets"`
	// +optional
	// Images is a list of images and tags that will be mirrored.  It is in the form
	// of <registry>/<image>:tag.
	Images []string `json:"images"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +k8s:defaulter-gen=true
// +kubebuilder:validation:Required
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Namespaced,shortName=mi,singular=mirror
// +kubebuilder:printcolumn:name="Images",type="integer",JSONPath=".status.totalImages",description="The number of total images managed by the object"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// Mirror is a resource defining images that will be mirrored to the local registry.
type Mirror struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MirrorSpec `json:"spec"`
	// +optional
	Status MirrorStatus `json:"status"`
}

type MirrorStatus struct {
	// +optional
	// TotalImages is the number of images that are being mirrored.
	TotalImages int `json:"totalImages"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

type MirrorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Mirror `json:"items"`
}
