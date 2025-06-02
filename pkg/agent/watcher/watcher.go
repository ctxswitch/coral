package watcher

import (
	"context"

	"ctx.sh/coral/pkg/agent/client"
	"ctx.sh/coral/pkg/agent/watcher/imagesync"
	"ctx.sh/coral/pkg/limiter"
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
	imageClient := client.New()
	if err := imageClient.Connect(ctx, opts.ContainerAddr); err != nil {
		return err
	}

	el := limiter.New(opts.MaxConcurrentPullers)

	if err := imagesync.SetupWithManager(mgr, &imagesync.Options{
		Limiter:                  el,
		MaxConcurrentReconcilers: opts.MaxConcurrentReconcilers,
		ImageClient:              imageClient,
		NodeName:                 opts.NodeName,
	}); err != nil {
		return err
	}

	return nil
}
