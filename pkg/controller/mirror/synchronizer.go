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
	"fmt"
	goruntime "runtime"

	utilauth "ctx.sh/coral/pkg/agent/watcher/imagesync"
	"ctx.sh/coral/pkg/util"
	"github.com/containers/image/v5/copy"
	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/signature"
	"github.com/containers/image/v5/types"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Synchronizer struct {
	secrets []corev1.Secret
	dst     string
	copyAll bool
}

func NewSynchronizer() *Synchronizer {
	return &Synchronizer{
		copyAll: false,
		secrets: make([]corev1.Secret, 0),
	}
}

func (s *Synchronizer) WithDestinationRegistry(dst string) *Synchronizer {
	s.dst = dst
	return s
}

func (s *Synchronizer) WithImagePullSecrets(secrets []corev1.Secret) *Synchronizer {
	s.secrets = append(s.secrets, secrets...)
	return s
}

func (s *Synchronizer) WithCopyAll(copyAll bool) *Synchronizer {
	s.copyAll = copyAll
	return s
}

func (s *Synchronizer) Copy(ctx context.Context, image string) error {
	logger := log.FromContext(ctx)

	srcImage := util.GetImageQualifiedName(util.DefaultSearchRegistry, image)
	dstImage := s.dst + "/" + util.ExtractImageName(image)

	logger.V(4).Info("copying mirror image", "src", srcImage, "dst", dstImage)

	// Create auth for source registry
	authProvider, err := utilauth.NewAuth(s.secrets)
	if err != nil {
		logger.Error(err, "failed to create auth")
		return fmt.Errorf("failed to create auth: %w", err)
	}

	// Create system context
	srcCtx := s.createSystemContext(ctx, srcImage, authProvider)
	dstCtx := s.createSystemContext(ctx, dstImage, authProvider)

	// Create source image reference
	srcRef, err := docker.ParseReference("//" + srcImage)
	if err != nil {
		return fmt.Errorf("failed to parse source reference: %w", err)
	}

	dstRef, err := docker.ParseReference("//" + dstImage)
	if err != nil {
		return fmt.Errorf("failed to parse destination reference: %w", err)
	}

	// TODO(rob): Make me configurable.
	policyCtx, err := signature.NewPolicyContext(&signature.Policy{
		Default: []signature.PolicyRequirement{signature.NewPRInsecureAcceptAnything()},
	})

	if err != nil {
		return fmt.Errorf("failed to create policy context: %w", err)
	}
	defer func() {
		_ = policyCtx.Destroy()
	}()

	// Determine image list selection based on configuration
	// Default to copying all architectures for backward compatibility
	imageListSelection := copy.CopyAllImages
	logMsg := "multi-arch image"

	if !s.copyAll {
		// Only copy system architecture - detect current runtime architecture
		imageListSelection = copy.CopySystemImage
		logMsg = "system architecture image"
		// Set system context to specify current platform
		srcCtx.OSChoice = goruntime.GOOS
		srcCtx.ArchitectureChoice = goruntime.GOARCH
	}

	// Perform the copy with configurable multi-arch support
	_, err = copy.Image(ctx, policyCtx, dstRef, srcRef, &copy.Options{
		SourceCtx:          srcCtx,
		DestinationCtx:     dstCtx,
		ImageListSelection: imageListSelection,
	})
	if err != nil {
		return fmt.Errorf("failed to copy %s: %w", logMsg, err)
	}

	return nil
}

// createSystemContext creates a system context for containers/image operations.
func (s *Synchronizer) createSystemContext(ctx context.Context, image string, authProvider *utilauth.Auth) *types.SystemContext {
	logger := ctrl.LoggerFrom(ctx)

	systemCtx := &types.SystemContext{
		// TODO: configurable
		DockerInsecureSkipTLSVerify: types.OptionalBoolTrue,
		// Disable Docker daemon - we're doing registry-to-registry operations (says claude, I dont believe it)
		DockerDaemonHost: "",
	}

	// Configure authentication if provided
	if authProvider != nil {
		logger.V(5).Info("configuring authentication for system context")

		// For now, we'll use a simple approach and configure auth for the source registry
		// This is simpler than the callback-based approach
		registry := util.ExtractImageHostname(image)
		authConfigs := authProvider.Lookup(registry)
		if len(authConfigs) > 0 {
			authConfig := authConfigs[0]
			logger.V(6).Info("found auth config for registry", "registry", registry, "username", authConfig.Username)

			systemCtx.DockerAuthConfig = &types.DockerAuthConfig{
				Username: authConfig.Username,
				Password: authConfig.Password,
			}
		}
	}

	return systemCtx
}
