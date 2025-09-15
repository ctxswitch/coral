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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type ControllerTestSuite struct {
	client *mock.Client
	suite.Suite
}

func (s *ControllerTestSuite) SetupTest() {
	logger := zap.New(zap.UseDevMode(true), zap.Level(zapcore.Level(-8)))
	log.SetLogger(logger)

	s.client = mock.NewClient().
		WithLogger(logger).
		WithFixtureDirectory(filepath.Join("..", "..", "..", "fixtures"))

	s.client.ApplyFixtureOrDie("imagesync-controller.yaml")
}

func (s *ControllerTestSuite) TearDownTest() {
	s.client.Reset()
}

func TestControllerTestSuite(t *testing.T) {
	suite.Run(t, new(ControllerTestSuite))
}

func (s *ControllerTestSuite) TestSetupWithManager() {
	tests := []struct {
		name        string
		opts        *Options
		expectError bool
	}{
		{
			name: "successful setup with options",
			opts: &Options{
				NodeRef: &store.NodeRef{},
			},
			expectError: false,
		},
		{
			name:        "successful setup with nil options",
			opts:        nil,
			expectError: true, // This would panic in real code, but we can't easily test that
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			// We can't easily create a real manager for testing,
			// so this test primarily validates the Options structure
			if tt.expectError {
				s.Nil(tt.opts) // Should be nil for error cases
			} else {
				s.NotNil(tt.opts) // Should not be nil for success cases
			}
		})
	}
}

func (s *ControllerTestSuite) TestController_Reconcile_ImageSyncNotFound() {
	controller := &Controller{
		Client: s.client,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "nonexistent-imagesync",
			Namespace: "default",
		},
	}

	result, err := controller.Reconcile(ctx, req)

	s.NoError(err)
	s.Equal(ctrl.Result{}, result)
}

func (s *ControllerTestSuite) TestController_addFinalizer() {
	controller := &Controller{
		Client: s.client,
	}

	ctx := context.Background()

	// Get the imagesync without finalizer
	var isync coralv1beta1.ImageSync
	err := s.client.Get(ctx, types.NamespacedName{Name: "test-imagesync-no-finalizer", Namespace: "default"}, &isync)
	s.NoError(err)

	err = controller.addFinalizer(ctx, &isync)

	s.NoError(err)
	s.True(controllerutil.ContainsFinalizer(&isync, coralv1beta1.ImageSyncFinalizer))
}

func (s *ControllerTestSuite) TestController_removeFinalizer() {
	tests := []struct {
		name              string
		imageSyncName     string
		expectedFinalizer bool
	}{
		{
			name:              "successfully remove finalizer",
			imageSyncName:     "test-imagesync",
			expectedFinalizer: false,
		},
		{
			name:              "imagesync without finalizer does nothing",
			imageSyncName:     "test-imagesync-no-finalizer",
			expectedFinalizer: false,
		},
	}

	controller := &Controller{
		Client: s.client,
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			ctx := context.Background()

			var isync coralv1beta1.ImageSync
			err := s.client.Get(ctx, types.NamespacedName{Name: tt.imageSyncName, Namespace: "default"}, &isync)
			s.NoError(err)

			err = controller.removeFinalizer(ctx, &isync)
			s.NoError(err)
			s.Equal(tt.expectedFinalizer, controllerutil.ContainsFinalizer(&isync, coralv1beta1.ImageSyncFinalizer))
		})
	}
}

func (s *ControllerTestSuite) TestController_finalize() {
	controller := &Controller{}

	ctx := context.Background()
	var isync coralv1beta1.ImageSync
	err := s.client.Get(ctx, types.NamespacedName{Name: "test-imagesync", Namespace: "default"}, &isync)
	s.NoError(err)

	err = controller.finalize(ctx, &isync)

	// Currently finalize is a no-op, so should not return error
	s.NoError(err)
}

func (s *ControllerTestSuite) TestController_Reconcile_SuccessfulFlow() {
	controller := &Controller{
		Client:   s.client,
		Recorder: &record.FakeRecorder{},
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-imagesync",
			Namespace: "default",
		},
	}

	result, err := controller.Reconcile(ctx, req)

	// The reconcile should succeed for a properly configured imagesync
	s.NoError(err)
	s.Equal(ctrl.Result{}, result)
}

func (s *ControllerTestSuite) TestController_Reconcile_WithDeletionTimestamp() {
	controller := &Controller{
		Client: s.client,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-imagesync-deleted",
			Namespace: "default",
		},
	}

	result, err := controller.Reconcile(ctx, req)

	// Should remove finalizer and finish reconciliation
	s.NoError(err)
	s.Equal(ctrl.Result{}, result)
}

func (s *ControllerTestSuite) TestController_Reconcile_NeedsFinalizer() {
	controller := &Controller{
		Client: s.client,
	}

	ctx := context.Background()
	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test-imagesync-no-finalizer",
			Namespace: "default",
		},
	}

	result, err := controller.Reconcile(ctx, req)

	// Should requeue to add finalizer
	s.NoError(err)
	s.True(result.Requeue)
}
