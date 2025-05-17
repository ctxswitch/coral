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
	ImageSyncFinalizer  = "imagesync.coral.ctx.sh/finalizer"
	MirrorFinalizer     = "mirror.coral.ctx.sh/finalizer"
	ImageSyncLabel      = "imagesync.coral.ctx.sh"
	ProcessedLabelName  = ImageSyncLabel + "/processed"
	ProcessedLabelValue = "true"
)

type NodeSelector struct {
	Key      string             `json:"key"`
	Operator selection.Operator `json:"operator"`
	Values   []string           `json:"values"`
}

// ImageSyncSpec is the spec for a Image resource.
type ImageSyncSpec struct {
	// +required
	// +listType=atomic
	// Images to fetch.
	Images []string `json:"images"`
	// +optional
	// +nullable
	// NodeSelector defines which nodes the image should be synced to.
	NodeSelector []NodeSelector `json:"nodeSelector,omitempty"`
	// +optional
	// +nullable
	// ImagePullSecrets is a list of secrets to use when pulling the image.
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
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
	ImageStateUpdating  ImageSyncState = "updating"
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
}

// ImageSyncImage contains details about an image that is being synced.
type ImageSyncImage struct {
	// +required
	// Name is the name of the image that the user has defined.
	Name string `json:"name,omitempty"`
	// +required
	// Image is the fully qualified image with tag that is being synced.  This represents what is
	// being parsed and represented on the agents, nodes, and webhooks.
	Image string `json:"image,omitempty"`
	// +required
	// Label is the digest of the image that is being synced.
	Label string `json:"label,omitempty"`
	// +optional
	// Available is the number of nodes that have the image available.
	Available int `json:"available,omitempty"`
	// +optional
	// Pending is the number of nodes that are pending image download.
	Pending int `json:"pending,omitempty"`
}

// ImageSyncStatus is the status for a WatchSet resource.
type ImageSyncStatus struct {
	// +optional
	// TotalNodes is the number of nodes that should have the image prefetched.
	TotalNodes int `json:"totalNodes,omitempty"`
	// +optional
	// TotalImages is the number of total images managed by the object.
	TotalImages int `json:"totalImages,omitempty"`
	// +optional
	// Condition is the current state of the images on the nodes.
	Condition ImageSyncCondition `json:"condition,omitempty"`
	// +optional
	// Data is a list of image data that will be used to track the images on the nodes.
	// Data []ImageSyncData `json:"data"`
	// +optional
	// The current state of the image sync resource.
	State ImageSyncState `json:"state,omitempty"`
	// +optional
	// Images is a list of images with label mappings.
	Images []ImageSyncImage `json:"images,omitempty"`
	// +optional
	// Revision is the hash of the current image sync resource.
	Revision string `json:"revision,omitempty"`
}

type MirrorSpec struct {
	// +required
	// LocalRegistry is the url to the local registry
	LocalRegistry string `json:"localRegistry,omitempty"`
	// +optional
	// +nullable
	// ImagePullSecrets is a list of secrets to use when pulling the image.
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets"`
	// +optional
	// Images is a list of images and tags that will be mirrored.  It is in the form
	// of <registry>/<image>:tag.
	Images []string `json:"images,omitempty"`
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
