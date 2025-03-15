package agent

import (
	"context"
	"ctx.sh/coral/pkg/agent/event"

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

func (w *Worker) Start(ctx context.Context, events <-chan event.Event) {
	log := logr.FromContextOrDiscard(ctx)
	log.V(6).Info("Starting worker")
	for evt := range events {
		if err := w.process(ctx, evt); err != nil {
			log.Error(err, "failed to process event")
		}
	}
}

func (w *Worker) process(ctx context.Context, e event.Event) error {
	log := logr.FromContextOrDiscard(ctx)
	log.V(6).Info(
		"Processing event",
		"name", e.Object.GetName(),
		"namespace", e.Object.GetNamespace(),
		"kind", e.Object.GetObjectKind().GroupVersionKind().Kind,
		"operation", e.GetOperationString(),
	)
	return nil
}
