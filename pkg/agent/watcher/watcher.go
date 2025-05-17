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

func SetupWithManager(ctx context.Context, mgr ctrl.Manager, opts *Options) (err error) {
	imageClient := image.New(mgr.GetClient(), opts.NodeName)
	if err = imageClient.Connect(ctx, opts.ContainerAddr); err != nil {
		return err
	}

	wq := queue.New(uint32(opts.MaxConcurrentPullers))

	if err = imagesync.SetupWithManager(mgr, &imagesync.Options{
		WorkQueue:                wq,
		MaxConcurrentReconcilers: opts.MaxConcurrentReconcilers,
		ImageClient:              imageClient,
		NodeName:                 opts.NodeName,
	}); err != nil {
		return
	}

	// TODO: set up polling node watcher.

	return
}
