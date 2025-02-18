package agent

import (
	"context"
	"github.com/go-logr/logr"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type WorkerOptions struct {
	Client               client.Client
	ImageServiceClient   runtime.ImageServiceClient
	RuntimeServiceClient runtime.RuntimeServiceClient
}

type Worker struct {
	client client.Client
	isc    runtime.ImageServiceClient
	rsc    runtime.RuntimeServiceClient
}

func NewWorker(opts *WorkerOptions) *Worker {
	return &Worker{
		client: opts.Client,
		isc:    opts.ImageServiceClient,
		rsc:    opts.RuntimeServiceClient,
	}
}

func (w *Worker) Start(ctx context.Context, events <-chan Event) {
	log := logr.FromContextOrDiscard(ctx)
	for evt := range events {
		if err := w.process(ctx, evt); err != nil {
			log.Error(err, "failed to process event")
		}
	}
}

func (w *Worker) process(ctx context.Context, e Event) error {
	return nil
}
