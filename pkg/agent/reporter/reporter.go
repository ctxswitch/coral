// Copyright 2025 Coral Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
