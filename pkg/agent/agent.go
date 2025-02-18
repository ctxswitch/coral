package agent

import (
	"context"
	"github.com/go-logr/logr"
	crun "k8s.io/cri-api/pkg/apis/runtime/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

type Options struct {
	WorkerCount          int
	Client               client.Client
	ImageServiceClient   crun.ImageServiceClient
	RuntimeServiceClient crun.RuntimeServiceClient
}

type Agent struct {
	workers int
	client  client.Client
}

func NewAgent(opts *Options) *Agent {
	return &Agent{
		client: opts.Client,
	}
}

func (a *Agent) Start(ctx context.Context) error {
	log := logr.FromContextOrDiscard(ctx)
	log.Info("starting agent")

	wg := sync.WaitGroup{}

	events := make(chan Event)
	for i := 0; i < a.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			w := NewWorker(&WorkerOptions{
				Client: a.client,
			})
			w.Start(ctx, events)
		}()
	}

	<-ctx.Done()

	log.Info("stopping agent")
	close(events)

	wg.Wait()
	return nil
}
