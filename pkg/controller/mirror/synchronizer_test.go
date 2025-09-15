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

package mirror

import (
	"context"
	"testing"

	"github.com/containers/image/v5/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewSynchronizer(t *testing.T) {
	s := NewSynchronizer()

	assert.NotNil(t, s)
	assert.False(t, s.copyAll, "copyAll should default to false")
	assert.Empty(t, s.secrets, "secrets should be empty by default")
	assert.Empty(t, s.dst, "destination should be empty by default")
}

func TestSynchronizer_WithDestinationRegistry(t *testing.T) {
	tests := []struct {
		name        string
		destination string
	}{
		{
			name:        "local registry with port",
			destination: "localhost:5000",
		},
		{
			name:        "custom registry",
			destination: "registry.example.com",
		},
		{
			name:        "empty destination",
			destination: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSynchronizer()
			result := s.WithDestinationRegistry(tt.destination)

			assert.Same(t, s, result, "should return the same instance for chaining")
			assert.Equal(t, tt.destination, s.dst, "destination should be set correctly")
		})
	}
}

func TestSynchronizer_WithImagePullSecrets(t *testing.T) {
	tests := []struct {
		name    string
		secrets []corev1.Secret
	}{
		{
			name: "single secret",
			secrets: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "test-secret",
						Namespace: "default",
					},
					Type: corev1.SecretTypeDockerConfigJson,
					Data: map[string][]byte{
						corev1.DockerConfigJsonKey: []byte(`{"auths":{"registry.io":{"auth":"dGVzdDp0ZXN0"}}}`),
					},
				},
			},
		},
		{
			name: "multiple secrets",
			secrets: []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret1",
						Namespace: "default",
					},
					Type: corev1.SecretTypeDockerConfigJson,
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "secret2",
						Namespace: "default",
					},
					Type: corev1.SecretTypeDockerConfigJson,
				},
			},
		},
		{
			name:    "empty secrets",
			secrets: []corev1.Secret{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSynchronizer()
			result := s.WithImagePullSecrets(tt.secrets)

			assert.Same(t, s, result, "should return the same instance for chaining")
			assert.Equal(t, len(tt.secrets), len(s.secrets), "should have correct number of secrets")

			// Test chaining multiple calls
			moreSecrets := []corev1.Secret{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "additional-secret",
					},
				},
			}
			s.WithImagePullSecrets(moreSecrets)
			assert.Equal(t, len(tt.secrets)+1, len(s.secrets), "should append secrets")
		})
	}
}

func TestSynchronizer_WithCopyAll(t *testing.T) {
	tests := []struct {
		name    string
		copyAll bool
	}{
		{
			name:    "copy all architectures",
			copyAll: true,
		},
		{
			name:    "copy system architecture only",
			copyAll: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSynchronizer()
			result := s.WithCopyAll(tt.copyAll)

			assert.Same(t, s, result, "should return the same instance for chaining")
			assert.Equal(t, tt.copyAll, s.copyAll, "copyAll should be set correctly")
		})
	}
}

func TestSynchronizer_ChainedConfiguration(t *testing.T) {
	secrets := []corev1.Secret{
		{
			ObjectMeta: metav1.ObjectMeta{
				Name: "test-secret",
			},
		},
	}

	s := NewSynchronizer().
		WithDestinationRegistry("localhost:5000").
		WithImagePullSecrets(secrets).
		WithCopyAll(true)

	assert.Equal(t, "localhost:5000", s.dst)
	assert.Equal(t, 1, len(s.secrets))
	assert.True(t, s.copyAll)
}

func TestSynchronizer_createSystemContext(t *testing.T) {
	tests := []struct {
		name         string
		image        string
		authProvider interface{} // Using interface{} since we can't easily create utilauth.Auth
		expectAuth   bool
	}{
		{
			name:         "docker.io image without auth",
			image:        "docker.io/library/nginx:latest",
			authProvider: nil,
			expectAuth:   false,
		},
		{
			name:         "custom registry without auth",
			image:        "registry.example.com/nginx:latest",
			authProvider: nil,
			expectAuth:   false,
		},
		{
			name:         "localhost registry without auth",
			image:        "localhost:5000/nginx:latest",
			authProvider: nil,
			expectAuth:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewSynchronizer()
			ctx := context.Background()

			// Since utilauth.Auth is complex to mock, we'll test the nil case
			systemCtx := s.createSystemContext(ctx, tt.image, nil)

			require.NotNil(t, systemCtx)
			assert.Equal(t, types.OptionalBoolTrue, systemCtx.DockerInsecureSkipTLSVerify)
			assert.Empty(t, systemCtx.DockerDaemonHost, "Docker daemon should be disabled")

			if !tt.expectAuth {
				assert.Nil(t, systemCtx.DockerAuthConfig)
			}
		})
	}
}

// Test error cases that don't require actual image operations.
func TestSynchronizer_Copy_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		setupSync     func() *Synchronizer
		image         string
		expectError   bool
		errorContains string
	}{
		{
			name:          "missing destination registry",
			setupSync:     NewSynchronizer,
			image:         "nginx",
			expectError:   true,
			errorContains: "", // This will fail at docker.ParseReference due to empty dst
		},
		{
			name: "valid configuration but will fail at image copy",
			setupSync: func() *Synchronizer {
				return NewSynchronizer().WithDestinationRegistry("localhost:5000")
			},
			image:       "nginx",
			expectError: true, // Will fail at actual image copy but that's expected
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.setupSync()
			ctx := context.Background()

			err := s.Copy(ctx, tt.image)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
