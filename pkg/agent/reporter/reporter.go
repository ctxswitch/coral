package reporter

import (
	"context"
	"time"

	"ctx.sh/coral/pkg/agent/client"
	"ctx.sh/coral/pkg/agent/reporter/image"
	ctrl "sigs.k8s.io/controller-runtime"
)

// TODO: These should be configurable.
const (
	// Endpoint is the default endpoint for the Coral service.
	Endpoint = "https://coral-webhook-service.coral-system.svc:443"
	// DefaultPollInterval is the default interval at which the reporter will poll and report changes.
	DefaultPollInterval = 5 * time.Second
)

type Options struct {
	ContainerAddr string
	NodeName      string
}

func SetupWithManager(ctx context.Context, mgr ctrl.Manager, opts *Options) error {
	imageClient := client.New()
	if err := imageClient.Connect(ctx, opts.ContainerAddr); err != nil {
		return err
	}

	if err := image.SetupWithManager(mgr, &image.Options{
		ImageClient:  imageClient,
		NodeName:     opts.NodeName,
		PollInterval: DefaultPollInterval,
		Endpoint:     Endpoint,
	}); err != nil {
		return err
	}
	return nil
}
