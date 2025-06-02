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

package controller

import (
	"crypto/tls"
	"os"

	"ctx.sh/coral/pkg/store"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"ctx.sh/coral/pkg/controller"
	"ctx.sh/coral/pkg/webhook"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	LeaderElectionID = "coral-controller-leader-lock"
)

type Controller struct {
	Certs              string
	LeaderElection     bool
	SkipInsecureVerify bool
	Namespace          string
	LogLevel           int8
}

func (c *Controller) RunE(cmd *cobra.Command, args []string) error {
	scheme := runtime.NewScheme()
	_ = coralv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	// TODO: more configurations to mirror bind flags.
	log := zap.New(
		zap.Level(zapcore.Level(c.LogLevel) * -1),
	)

	ctx := ctrl.SetupSignalHandler()
	ctrl.SetLogger(log)

	// TODO: controller-runtime now has a cert-watcher.  Set this up for the webhooks

	log.Info("starting coral controller", "controller", c)

	hookServer := webhook.NewServer(webhook.ServerOptions{
		Port:    9443,
		CertDir: c.Certs,
		// TODO: One of these causes an error about 'client didn't provide a certificate'
		// Look at these settings in more detail later.
		// CertName:     DefaultCertName,
		// KeyName:      DefaultKeyName,
		// ClientCAName: DefaultClientCAName,
		TLSOpts: []func(*tls.Config){
			func(config *tls.Config) {
				config.InsecureSkipVerify = c.SkipInsecureVerify
			},
		},
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                        scheme,
		LeaderElection:                c.LeaderElection,
		LeaderElectionID:              LeaderElectionID,
		LeaderElectionReleaseOnCancel: true,
		WebhookServer:                 hookServer,
	})

	if err != nil {
		log.Error(err, "unable to initialize manager")
		os.Exit(1)
	}

	nodeRef := store.NewNodeRef()

	// Set up controllers
	if err = controller.SetupWithManager(mgr, &controller.Options{
		NodeRef: nodeRef,
	}); err != nil {
		log.Error(err, "unable to setup controllers")
		os.Exit(1)
	}

	// Set up webhooks
	if err = webhook.SetupWebhooksWithManager(mgr, &webhook.Options{
		NodeRef: nodeRef,
	}); err != nil {
		log.Error(err, "unable to setup webhooks")
		os.Exit(1)
	}

	// Start the manager process
	log.Info("starting manager")

	return mgr.Start(ctx)
}
