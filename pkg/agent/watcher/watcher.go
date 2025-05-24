package watcher

import (
	"context"

	"ctx.sh/coral/pkg/agent/image"
	"ctx.sh/coral/pkg/agent/watcher/imagesync"
	"ctx.sh/coral/pkg/queue"
	ctrl "sigs.k8s.io/controller-runtime"
)

type Options struct {
	ContainerAddr            string
	MaxConcurrentReconcilers int
	MaxConcurrentPullers     int
	NodeName                 string
}

type Watcher struct{}

func SetupWithManager(ctx context.Context, mgr ctrl.Manager, opts *Options) error {
	imageClient := image.New()
	if err := imageClient.Connect(ctx, opts.ContainerAddr); err != nil {
		return err
	}

	wq := queue.New(opts.MaxConcurrentPullers)

	if err := imagesync.SetupWithManager(mgr, &imagesync.Options{
		WorkQueue:                wq,
		MaxConcurrentReconcilers: opts.MaxConcurrentReconcilers,
		ImageClient:              imageClient,
		NodeName:                 opts.NodeName,
	}); err != nil {
		return err
	}

	// TODO: set up polling node watcher.

	return nil
}
