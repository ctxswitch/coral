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

package imagesync

import (
	"context"
	"path/filepath"
	"testing"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/mock"
	"ctx.sh/coral/pkg/store"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type StatusUpdaterTestSuite struct {
	client  *mock.Client
	nodeRef *store.NodeRef
	suite.Suite
}

func (s *StatusUpdaterTestSuite) SetupTest() {
	logger := zap.New(zap.UseDevMode(true), zap.Level(zapcore.Level(-8)))
	log.SetLogger(logger)

	s.client = mock.NewClient().
		WithLogger(logger).
		WithFixtureDirectory(filepath.Join("..", "..", "..", "fixtures"))

	s.client.ApplyFixtureOrDie("imagesync-controller.yaml")
	s.client.ApplyFixtureOrDie("nodes-controller.yaml")

	// Setup nodeRef for testing
	s.nodeRef = store.NewNodeRef()
}

func (s *StatusUpdaterTestSuite) TearDownTest() {
	// Reset nodeRef for clean state between tests
	s.nodeRef = store.NewNodeRef()
}

func TestStatusUpdaterTestSuite(t *testing.T) {
	suite.Run(t, new(StatusUpdaterTestSuite))
}

func (s *StatusUpdaterTestSuite) TestNewStatusUpdater() {
	updater := NewStatusUpdater(s.client, s.nodeRef)

	s.NotNil(updater)
	s.Equal(s.client, updater.Client)
	s.Equal(s.nodeRef, updater.nodeRef)
	s.NotNil(updater.stopCh)
}

func (s *StatusUpdaterTestSuite) TestStatusUpdater_NeedLeaderElection() {
	updater := &StatusUpdater{}
	s.True(updater.NeedLeaderElection())
}

func (s *StatusUpdaterTestSuite) TestStatusUpdater_filterNodes() {
	tests := []struct {
		name      string
		nodes     []corev1.Node
		selectors []coralv1beta1.NodeSelector
		expected  int
	}{
		{
			name: "no selectors - all nodes match",
			nodes: []corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{Name: "node1"}},
				{ObjectMeta: metav1.ObjectMeta{Name: "node2"}},
			},
			selectors: []coralv1beta1.NodeSelector{},
			expected:  2,
		},
		{
			name: "label selector matches some nodes",
			nodes: []corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "node1",
					Labels: map[string]string{"env": "prod"},
				}},
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "node2",
					Labels: map[string]string{"env": "dev"},
				}},
			},
			selectors: []coralv1beta1.NodeSelector{
				{
					Key:      "env",
					Operator: selection.In,
					Values:   []string{"prod"},
				},
			},
			expected: 1,
		},
		{
			name: "no nodes match selector",
			nodes: []corev1.Node{
				{ObjectMeta: metav1.ObjectMeta{
					Name:   "node1",
					Labels: map[string]string{"env": "prod"},
				}},
			},
			selectors: []coralv1beta1.NodeSelector{
				{
					Key:      "env",
					Operator: selection.In,
					Values:   []string{"staging"},
				},
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			updater := &StatusUpdater{}
			filtered := updater.filterNodes(tt.nodes, tt.selectors)
			s.Equal(tt.expected, len(filtered))
		})
	}
}

func (s *StatusUpdaterTestSuite) TestStatusUpdater_matches() {
	tests := []struct {
		name      string
		labels    map[string]string
		selectors []coralv1beta1.NodeSelector
		expected  bool
	}{
		{
			name:      "empty selectors always match",
			labels:    map[string]string{"key": "value"},
			selectors: []coralv1beta1.NodeSelector{},
			expected:  true,
		},
		{
			name:   "single selector matches",
			labels: map[string]string{"env": "prod"},
			selectors: []coralv1beta1.NodeSelector{
				{
					Key:      "env",
					Operator: selection.In,
					Values:   []string{"prod"},
				},
			},
			expected: true,
		},
		{
			name:   "single selector does not match",
			labels: map[string]string{"env": "dev"},
			selectors: []coralv1beta1.NodeSelector{
				{
					Key:      "env",
					Operator: selection.In,
					Values:   []string{"prod"},
				},
			},
			expected: false,
		},
		{
			name:   "NotIn operator works",
			labels: map[string]string{"env": "prod"},
			selectors: []coralv1beta1.NodeSelector{
				{
					Key:      "env",
					Operator: selection.NotIn,
					Values:   []string{"dev", "staging"},
				},
			},
			expected: true,
		},
		{
			name:   "invalid operator returns false",
			labels: map[string]string{"env": "prod"},
			selectors: []coralv1beta1.NodeSelector{
				{
					Key:      "env",
					Operator: selection.Operator("InvalidOp"),
					Values:   []string{"prod"},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			updater := &StatusUpdater{}
			result := updater.matches(tt.labels, tt.selectors)
			s.Equal(tt.expected, result)
		})
	}
}

func (s *StatusUpdaterTestSuite) TestStatusUpdater_updateStatus_NoChange() {
	// Get an ImageSync from fixtures and set up status
	ctx := context.Background()
	var isync coralv1beta1.ImageSync
	err := s.client.Get(ctx, types.NamespacedName{Name: "test-imagesync", Namespace: "default"}, &isync)
	s.NoError(err)

	// Set status to test no-change scenario
	isync.Status = coralv1beta1.ImageSyncStatus{
		TotalNodes:  2,
		TotalImages: 1,
		Condition: coralv1beta1.ImageSyncCondition{
			Available: 1,
			Pending:   1,
		},
	}

	updater := &StatusUpdater{
		Client: s.client,
	}

	// Same status - should not update
	err = updater.updateStatus(ctx, &isync, isync.Status)
	s.NoError(err)
}

func (s *StatusUpdaterTestSuite) TestStatusUpdater_updateStatus_WithChange() {
	ctx := context.Background()
	var isync coralv1beta1.ImageSync
	err := s.client.Get(ctx, types.NamespacedName{Name: "test-imagesync", Namespace: "default"}, &isync)
	s.NoError(err)

	updater := &StatusUpdater{
		Client: s.client,
	}

	newStatus := coralv1beta1.ImageSyncStatus{
		TotalNodes:  2,
		TotalImages: 1,
		Condition: coralv1beta1.ImageSyncCondition{
			Available: 1,
			Pending:   1,
		},
	}

	err = updater.updateStatus(ctx, &isync, newStatus)
	s.NoError(err)
	s.Equal(newStatus.TotalNodes, isync.Status.TotalNodes)
	s.Equal(newStatus.TotalImages, isync.Status.TotalImages)
	s.NotNil(isync.Status.LastUpdated)
}

func (s *StatusUpdaterTestSuite) TestStatusUpdater_update_NoImageSync() {
	// Reset client to have no ImageSync resources
	s.client.Reset()

	updater := &StatusUpdater{
		Client:  s.client,
		nodeRef: s.nodeRef,
	}

	ctx := context.Background()
	err := updater.update(ctx)
	s.NoError(err)
}

func (s *StatusUpdaterTestSuite) TestStatusUpdater_update_WithImageSync() {
	// Set up nodeRef with images
	s.nodeRef.AddImages("node1", []string{"docker.io/library/nginx:latest", "docker.io/library/redis:latest"})
	s.nodeRef.AddImages("node2", []string{"docker.io/library/nginx:latest"})

	updater := &StatusUpdater{
		Client:  s.client,
		nodeRef: s.nodeRef,
	}

	ctx := context.Background()
	err := updater.update(ctx)
	s.NoError(err)

	// Verify the imagesync status was updated
	var updatedImageSync coralv1beta1.ImageSync
	err = s.client.Get(ctx, types.NamespacedName{Name: "test-imagesync-multi-image", Namespace: "default"}, &updatedImageSync)
	s.NoError(err)

	s.Equal(2, updatedImageSync.Status.TotalNodes)
	s.Equal(2, updatedImageSync.Status.TotalImages)
	s.Equal(2, len(updatedImageSync.Status.Images))

	// Find nginx image status
	var nginxStatus *coralv1beta1.ImageSyncImage
	for i := range updatedImageSync.Status.Images {
		if updatedImageSync.Status.Images[i].Image == "docker.io/library/nginx:latest" {
			nginxStatus = &updatedImageSync.Status.Images[i]
			break
		}
	}
	s.NotNil(nginxStatus)
	s.Equal(2, nginxStatus.Available) // Available on both nodes
	s.Equal(0, nginxStatus.Pending)   // Pending = 2 nodes - 2 available = 0

	// Find redis image status
	var redisStatus *coralv1beta1.ImageSyncImage
	for i := range updatedImageSync.Status.Images {
		if updatedImageSync.Status.Images[i].Image == "docker.io/library/redis:latest" {
			redisStatus = &updatedImageSync.Status.Images[i]
			break
		}
	}
	s.NotNil(redisStatus)
	s.Equal(1, redisStatus.Available) // Available on node1 only
	s.Equal(1, redisStatus.Pending)   // Pending on node2

	// Overall condition should be based on the minimum (redis is the limiting factor)
	s.Equal(1, updatedImageSync.Status.Condition.Available) // Min of nginx(2) and redis(1) = 1
	s.Equal(1, updatedImageSync.Status.Condition.Pending)   // 2 total nodes - 1 available = 1 pending
}

func (s *StatusUpdaterTestSuite) TestStatusUpdater_update_NoMatchingNodes() {
	updater := &StatusUpdater{
		Client:  s.client,
		nodeRef: s.nodeRef,
	}

	ctx := context.Background()
	err := updater.update(ctx)
	s.NoError(err)

	// Verify the imagesync status shows matching prod nodes (node1, node2 have env: prod)
	var updatedImageSync coralv1beta1.ImageSync
	err = s.client.Get(ctx, types.NamespacedName{Name: "test-imagesync-no-matching-nodes", Namespace: "default"}, &updatedImageSync)
	s.NoError(err)

	// This ImageSync has a prod selector and node1, node2 have env: prod in fixtures
	s.Equal(2, updatedImageSync.Status.TotalNodes)
	s.Equal(1, updatedImageSync.Status.TotalImages)
	// No images in nodeRef, so all should be pending
	s.Equal(0, updatedImageSync.Status.Condition.Available)
	s.Equal(2, updatedImageSync.Status.Condition.Pending)
}
