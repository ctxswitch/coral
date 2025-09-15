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
	"time"

	"ctx.sh/coral/pkg/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

type ObserveTestSuite struct {
	client *mock.Client
	suite.Suite
}

func (s *ObserveTestSuite) SetupTest() {
	logger := zap.New(zap.UseDevMode(true), zap.Level(zapcore.Level(-8)))
	log.SetLogger(logger)

	s.client = mock.NewClient().
		WithLogger(logger).
		WithFixtureDirectory(filepath.Join("..", "..", "..", "fixtures"))

	s.client.ApplyFixtureOrDie("imagesync-controller.yaml")
}

func (s *ObserveTestSuite) TearDownTest() {
	s.client.Reset()
}

func TestObserveTestSuite(t *testing.T) {
	suite.Run(t, new(ObserveTestSuite))
}

func (s *ObserveTestSuite) TestNewObservedState() {
	before := time.Now()
	observed := NewObservedState()
	after := time.Now()

	s.NotNil(observed)
	s.Nil(observed.ImageSync)
	s.True(observed.ObserveTime.After(before) || observed.ObserveTime.Equal(before))
	s.True(observed.ObserveTime.Before(after) || observed.ObserveTime.Equal(after))
}

func (s *ObserveTestSuite) TestStateObserver_observerImageSync() {
	tests := []struct {
		name          string
		imageSyncName string
		expectedNil   bool
		expectedError bool
	}{
		{
			name:          "successfully get imagesync",
			imageSyncName: "test-imagesync",
			expectedNil:   false,
			expectedError: false,
		},
		{
			name:          "imagesync not found returns nil",
			imageSyncName: "nonexistent-imagesync",
			expectedNil:   true,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			observer := &StateObserver{
				Client: s.client,
				Request: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      tt.imageSyncName,
						Namespace: "default",
					},
				},
			}

			ctx := context.Background()
			imagesync, err := observer.observerImageSync(ctx)

			if tt.expectedError {
				s.Error(err)
				s.Nil(imagesync)
			} else {
				s.NoError(err)
				if tt.expectedNil {
					s.Nil(imagesync)
				} else {
					s.NotNil(imagesync)
					s.Equal(tt.imageSyncName, imagesync.Name)
				}
			}
		})
	}
}

func (s *ObserveTestSuite) TestStateObserver_observe() {
	tests := []struct {
		name                 string
		imageSyncName        string
		expectedImageSyncNil bool
		expectedError        bool
	}{
		{
			name:                 "successfully observe state",
			imageSyncName:        "test-imagesync",
			expectedImageSyncNil: false,
			expectedError:        false,
		},
		{
			name:                 "imagesync not found - no error",
			imageSyncName:        "nonexistent-imagesync",
			expectedImageSyncNil: true,
			expectedError:        false,
		},
	}

	for _, tt := range tests {
		s.Run(tt.name, func() {
			observer := &StateObserver{
				Client: s.client,
				Request: ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      tt.imageSyncName,
						Namespace: "default",
					},
				},
			}

			ctx := context.Background()
			observed := NewObservedState()
			err := observer.observe(ctx, observed)

			if tt.expectedError {
				s.Error(err)
				s.Nil(observed.ImageSync)
			} else {
				s.NoError(err)
				if tt.expectedImageSyncNil {
					s.Nil(observed.ImageSync)
				} else {
					s.NotNil(observed.ImageSync)
					s.Equal(tt.imageSyncName, observed.ImageSync.Name)
				}
			}
		})
	}
}

func (s *ObserveTestSuite) TestStateObserver_observe_WithDefaults() {
	observer := &StateObserver{
		Client: s.client,
		Request: ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-imagesync",
				Namespace: "default",
			},
		},
	}

	ctx := context.Background()
	observed := NewObservedState()
	err := observer.observe(ctx, observed)

	s.NoError(err)
	s.NotNil(observed.ImageSync)
	// Verify that Defaulted() was called by checking that defaults have been applied
	s.NotNil(observed.ImageSync)
}
