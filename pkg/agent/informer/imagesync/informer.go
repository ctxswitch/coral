package imagesync

import (
	"context"
	"ctx.sh/coral/pkg/agent"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"os"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sync"
)

type Informer struct {
	stopCh   chan struct{}
	stopOnce sync.Once

	cache.Cache
}

func SetupWithManager(mgr ctrl.Manager) *Informer {
	return &Informer{
		Cache: mgr.GetCache(),
	}
}

func (i *Informer) Start(ctx context.Context, events chan<- agent.Event) error {
	logger := log.FromContext(ctx).WithName("informer.imagesync")
	i.stopCh = make(chan struct{})

	informer, err := i.GetInformerForKind(ctx, schema.GroupVersionKind{
		Group:   "coral.ctx.sh",
		Version: "v1beta1",
		Kind:    "ImageSync",
	})
	if err != nil {
		return err
	}

	_, err = informer.AddEventHandler(&Handler{})
	if err != nil {
		return err
	}

	if !i.Cache.WaitForCacheSync(ctx) {
		return ctx.Err()
	}

	go func() {
		if err := i.Cache.Start(ctx); err != nil {
			logger.Error(err, "failed to start cache")
			os.Exit(1)
		}

		// Once we return from the informer start, close the stop
		// channel.  The stop function will block until closed.  Cheap
		// way to make sure that we don't close the event channels in the
		// parent before the informers have shut down.
		i.stopOnce.Do(func() {
			close(i.stopCh)
		})
		logger.Info("imagesync informer stopped")
	}()

	return nil
}

func (i *Informer) Stop() {
	// Just hang around until the informer has exited.
	<-i.stopCh
}
