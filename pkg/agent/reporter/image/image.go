package image

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"connectrpc.com/connect"
	"ctx.sh/coral/pkg/agent/client"
	coralv1beta1 "ctx.sh/coral/pkg/gen/coral/v1beta1"
	"ctx.sh/coral/pkg/gen/coral/v1beta1/coralv1beta1connect"
	"golang.org/x/net/http2"
	ctrl "sigs.k8s.io/controller-runtime"
)

var (
	DefaultBackoff = wait.Backoff{ //nolint:gochecknoglobals
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Steps:    10,
		Jitter:   0.4,
	}
)

type Options struct {
	ImageClient  client.ImageClient
	NodeName     string
	PollInterval time.Duration
	Endpoint     string
}

type Image struct {
	ImageClient  client.ImageClient
	NodeName     string
	PollInterval time.Duration
	Endpoint     string
}

func SetupWithManager(mgr ctrl.Manager, opts *Options) error {
	img := &Image{
		ImageClient:  opts.ImageClient,
		NodeName:     opts.NodeName,
		Endpoint:     opts.Endpoint,
		PollInterval: opts.PollInterval,
	}

	return mgr.Add(img)
}

func (i *Image) NeedLeaderElection() bool {
	return true
}

func (i *Image) Start(ctx context.Context) error {
	ticker := time.NewTicker(i.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := i.run(ctx); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "Failed to report images to coral service")
			}
		}
	}
}

func (i *Image) run(ctx context.Context) error {
	// TODO: If we don't have any matching imagesyncs, we don't need to actually send.

	hc := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP: true,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: true, //nolint:gosec
			},
		},
	}

	conn := coralv1beta1connect.NewCoralServiceClient(hc, i.Endpoint)
	images, err := i.ImageClient.List(ctx)
	if err != nil {
		return err
	}

	count := 0
	var resp *connect.Response[coralv1beta1.ReporterResponse]
	err = wait.ExponentialBackoffWithContext(ctx, DefaultBackoff, func(context.Context) (bool, error) {
		count++
		if resp, err = conn.Reporter(ctx, connect.NewRequest(&coralv1beta1.ReporterRequest{
			Image: images,
			Node:  i.NodeName,
		})); err != nil {
			if count < DefaultBackoff.Steps {
				return false, nil
			}
			return false, err
		}

		return true, nil
	})
	if err != nil {
		return err
	}

	if resp.Msg.Status != coralv1beta1.ReporterStatus_OK {
		return fmt.Errorf("%s", resp.Msg.Status.String())
	}

	return nil
}
