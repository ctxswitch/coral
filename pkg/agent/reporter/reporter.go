package reporter

import (
	"context"
	"crypto/tls"
	"time"

	"ctx.sh/coral/pkg/agent/client"
	"ctx.sh/coral/pkg/agent/reporter/image"
	ctrl "sigs.k8s.io/controller-runtime"
)

// TODO: These should be configurable.
const (
	// Host is the default endpoint for the Coral service.
	Host = "https://coral-webhook-service.coral-system.svc"
	// DefaultPollInterval is the default interval at which the reporter will poll and report changes.
	DefaultPollInterval = 5 * time.Second
)

type Options struct {
	ContainerAddr      string
	NodeName           string
	Host               string
	Port               int
	CertDir            string
	CertName           string
	KeyName            string
	ClientCAName       string
	TLSOpts            []func(*tls.Config)
	InsecureSkipVerify bool
}

func SetupWithManager(ctx context.Context, mgr ctrl.Manager, opts *Options) error {
	imageClient := client.New()
	if err := imageClient.Connect(ctx, opts.ContainerAddr); err != nil {
		return err
	}

	if err := image.SetupWithManager(mgr, image.Options{
		ImageClient:  imageClient,
		NodeName:     opts.NodeName,
		PollInterval: DefaultPollInterval,
		Host:         opts.Host,
		CertName:     opts.CertName,
		KeyName:      opts.KeyName,
		CertDir:      opts.CertDir,
		TLSOpts:      opts.TLSOpts,
	}); err != nil {
		return err
	}
	return nil
}
