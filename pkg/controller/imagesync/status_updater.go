package imagesync

import (
	"context"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"time"

	// "sigs.k8s.io/controller-runtime/pkg/manager"
	"sync"
)

// StatusUpdater is a process that runs in the background to update the status of the
// imagesync on the required nodes.
type StatusUpdater struct {
	stopCh   chan struct{}
	stopOnce sync.Once
	client.Client
}

// Start starts the status updater process.
func (su *StatusUpdater) Start(ctx context.Context) error {
	timer := time.NewTicker(5 * time.Second)

	// TODO: Add workers.

	for {
		select {
		case <-timer.C:
			// TODO: Update the status of the imagesyncs.
		case <-ctx.Done():
			timer.Stop()
		case <-su.stopCh:
			return nil
		}
	}
}

func (su *StatusUpdater) Stop() {
	su.stopOnce.Do(func() {
		close(su.stopCh)
	})
}
