package registry

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"

	"github.com/sirupsen/logrus"

	"github.com/distribution/distribution/v3/registry"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/inmemory"
	_ "github.com/distribution/distribution/v3/registry/storage/driver/s3-aws"
	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// +kubebuilder:skip

// +kubebuilder:skip
type Registry struct {
	Options  *Options
	registry *registry.Registry
	mu       sync.RWMutex
}

func SetupWebhookWithManager(ctx context.Context, mgr ctrl.Manager, opts *Options) error {
	reg := &Registry{
		Options: opts,
	}

	return mgr.Add(reg)
}

// NeedLeaderElection indicates whether the registry service needs leader election.
func (r *Registry) NeedLeaderElection() bool {
	return false
}

// Start is the entry point for the registry service.
func (r *Registry) Start(ctx context.Context) error {
	log := ctrl.LoggerFrom(ctx)

	r.mu.Lock()

	// Apply defaults to any unset options
	r.Options.setDefaults()

	config := NewConfiguration(r.Options)

	log.Info("starting registry service")

	if !r.Options.EnableRegistryLogging {
		logrus.SetOutput(io.Discard)
	}

	// Create the distribution registry
	reg, err := registry.NewRegistry(ctx, config.RegistryConfig())
	if err != nil {
		r.mu.Unlock()
		log.Error(err, "failed to create registry instance")
		return fmt.Errorf("failed to create registry: %w", err)
	}

	r.registry = reg
	log.Info("registry service created successfully, starting with graceful shutdown support")

	r.mu.Unlock()

	// Start registry in a goroutine for graceful shutdown
	errChan := make(chan error, 1)

	go func() {
		if err := r.registry.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error(err, "registry service failed")
			errChan <- fmt.Errorf("registry service failed: %w", err)
			return
		}
		errChan <- nil
	}()

	// Wait for context cancellation or error
	select {
	case <-ctx.Done():
		log.Info("shutting down registry service")
		return r.shutdown(log)
	case err := <-errChan:
		return err
	}
}

// shutdown gracefully shuts down the registry service.
func (r *Registry) shutdown(log logr.Logger) error {
	log.Info("initiating graceful shutdown of registry service")

	// TODO: The distribution registry doesn't expose direct shutdown methods
	// For now, we rely on the context cancellation to stop the service
	// In a future enhancement, we might need to implement a custom server wrapper

	log.Info("registry service shutdown completed")
	return nil
}
