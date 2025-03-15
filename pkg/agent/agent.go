package agent

import (
	"context"
	"ctx.sh/coral/pkg/agent/event"
	"ctx.sh/coral/pkg/agent/informer/imagesync"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sync"

	"github.com/go-logr/logr"
	crun "k8s.io/cri-api/pkg/apis/runtime/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Options struct {
	Workers              int
	ImageServiceClient   crun.ImageServiceClient
	RuntimeServiceClient crun.RuntimeServiceClient
}

type Agent struct {
	workers              int
	client               client.Client
	cache                cache.Cache
	imageServiceClient   crun.ImageServiceClient
	runtimeServiceClient crun.RuntimeServiceClient
}

func SetupWithManager(mgr ctrl.Manager, opts *Options) *Agent {
	return &Agent{
		workers:              opts.Workers,
		client:               mgr.GetClient(),
		cache:                mgr.GetCache(),
		imageServiceClient:   opts.ImageServiceClient,
		runtimeServiceClient: opts.RuntimeServiceClient,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("starting agent")

	events := make(chan event.Event)

	wg := sync.WaitGroup{}

	log.Info("starting workers")
	for i := 0; i < a.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := NewWorker(&WorkerOptions{
				Client: a.client,
				// TODO: add image services and other dependencies.
			})
			wctx := logr.NewContext(ctx, log.WithValues("worker", i))
			w.Start(wctx, events)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		imageSyncInformer := imagesync.Setup(&imagesync.Options{
			Cache: a.cache,
		})
		if err := imageSyncInformer.Start(ctx, events); err != nil {
			log.Error(err, "failed to start image sync informer")
			// Stop the world...
		}
	}()

	<-ctx.Done()

	log.Info("stopping agent")

	close(events)

	wg.Wait()
	return nil
}
