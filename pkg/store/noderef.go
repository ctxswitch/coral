// Copyright 2025 Coral Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import "sync"

type Images struct {
	images map[string]bool
}

type NodeRef struct {
	refs map[string]*Images
	sync.Mutex
}

func NewNodeRef() *NodeRef {
	return &NodeRef{
		refs: make(map[string]*Images),
	}
}

func (nr *NodeRef) AddImages(node string, images []string) {
	nr.Lock()
	defer nr.Unlock()

	nr.resetNode(node)
	for _, image := range images {
		nr.addImage(node, image)
	}
}

// AddImage adds an image to a node.
func (nr *NodeRef) addImage(nodeName, imageName string) {
	if _, exists := nr.refs[nodeName]; !exists {
		nr.refs[nodeName] = &Images{images: make(map[string]bool)}
	}
	nr.refs[nodeName].images[imageName] = true
}

// HasImage checks if a node has a specific image.
func (nr *NodeRef) HasImage(nodeName, imageName string) bool {
	nr.Lock()
	defer nr.Unlock()

	if images, exists := nr.refs[nodeName]; exists {
		if available, ok := images.images[imageName]; ok {
			return available
		}
	}

	return false
}

// GetNodes returns a list of node names that are being tracked.
func (nr *NodeRef) GetNodes() []string {
	nr.Lock()
	defer nr.Unlock()

	nodes := make([]string, 0, len(nr.refs))
	for nodeName := range nr.refs {
		nodes = append(nodes, nodeName)
	}
	return nodes
}

// GetAvailableImages returns a list of images that are available for a given node.
func (nr *NodeRef) GetAvailableImages(nodeName string) []string {
	nr.Lock()
	defer nr.Unlock()

	if images, exists := nr.refs[nodeName]; exists {
		imageList := make([]string, 0)
		for k, v := range images.images {
			if v {
				imageList = append(imageList, k)
			}
		}
		return imageList
	}
	return nil
}

// ResetNode resets the value of "available" for a node's image.
func (nr *NodeRef) resetNode(nodeName string) {
	if _, exists := nr.refs[nodeName]; exists {
		for k, v := range nr.refs[nodeName].images {
			if v {
				nr.refs[nodeName].images[k] = false
			}
		}
	}
}
