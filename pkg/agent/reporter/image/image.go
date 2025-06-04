package image

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/certwatcher"
	"time"

	"k8s.io/apimachinery/pkg/util/wait"

	"connectrpc.com/connect"
	"ctx.sh/coral/pkg/agent/client"
	coralv1beta1 "ctx.sh/coral/pkg/gen/coral/v1beta1"
	"ctx.sh/coral/pkg/gen/coral/v1beta1/coralv1beta1connect"
	"golang.org/x/net/http2"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	DefaultPort = 9443
)

var (
	DefaultBackoff = wait.Backoff{ //nolint:gochecknoglobals
		Duration: 500 * time.Millisecond,
		Factor:   1.5,
		Steps:    10,
		Jitter:   0.4,
	}
)

// TODO: Client and server options overlap and should be merged into a single shared struct.

type Options struct {
	ImageClient        client.ImageClient
	NodeName           string
	PollInterval       time.Duration
	Host               string
	CertName           string
	KeyName            string
	CertDir            string
	ClientCAName       string
	TLSOpts            []func(*tls.Config)
	InsecureSkipVerify bool
}

type Image struct {
	ImageClient  client.ImageClient
	NodeName     string
	PollInterval time.Duration
	Options      Options
}

func SetupWithManager(mgr ctrl.Manager, opts Options) error {
	img := &Image{
		ImageClient:  opts.ImageClient,
		NodeName:     opts.NodeName,
		PollInterval: opts.PollInterval,
		Options:      opts,
	}

	return mgr.Add(img)
}

func (i *Image) NeedLeaderElection() bool {
	return true
}

func (i *Image) Start(ctx context.Context) error {
	// TODO: If we don't have any matching imagesyncs, we don't need to actually send.
	log := ctrl.LoggerFrom(ctx)

	cfg := &tls.Config{
		InsecureSkipVerify: i.Options.InsecureSkipVerify, //nolint:gosec
	}

	for _, op := range i.Options.TLSOpts {
		op(cfg)
	}

	if cfg.GetCertificate == nil {
		certPath := filepath.Join(i.Options.CertDir, i.Options.CertName)
		keyPath := filepath.Join(i.Options.CertDir, i.Options.KeyName)
		certWatcher, err := certwatcher.New(certPath, keyPath)
		if err != nil {
			return err
		}
		cfg.GetCertificate = certWatcher.GetCertificate

		go func() {
			if err := certWatcher.Start(ctx); err != nil {
				log.Error(err, "certificate watcher error")
			}
		}()
	}

	hc := &http.Client{
		Transport: &http2.Transport{
			AllowHTTP:       true,
			TLSClientConfig: cfg,
		},
	}

	ticker := time.NewTicker(i.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if err := i.run(ctx, hc); err != nil {
				ctrl.LoggerFrom(ctx).Error(err, "Failed to report images to coral service")
			}
		}
	}
}

func (i *Image) run(ctx context.Context, hc *http.Client) error {
	conn := coralv1beta1connect.NewCoralServiceClient(hc, i.Options.Host)
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
