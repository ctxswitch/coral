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

package agent

import (
	"crypto/tls"
	"os"
	"time"

	"ctx.sh/coral/pkg/agent/reporter"

	"ctx.sh/coral/pkg/agent/watcher"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	ConnectionTimeout  time.Duration = 30 * time.Second
	MaxCallRecvMsgSize int           = 1024 * 1024 * 32
)

type Agent struct {
	ContainerdAddr           string
	LogLevel                 int8
	MaxConcurrentReconcilers int
	MaxConcurrentPullers     int
	Host                     string
	CertDir                  string
	CertName                 string
	KeyName                  string
	SkipInsecureVerify       bool
	ClientCAName             string
}

func (a *Agent) RunE(cmd *cobra.Command, args []string) error {
	scheme := runtime.NewScheme()
	_ = coralv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)

	log := zap.New(
		zap.Level(zapcore.Level(a.LogLevel) * -1),
	)

	ctx := ctrl.SetupSignalHandler()
	ctrl.SetLogger(log)

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Error(nil, "NODE_NAME must be set in the environment.")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
	})
	if err != nil {
		log.Error(err, "unable to initialize manager")
		return err
	}

	if err = watcher.SetupWithManager(ctx, mgr, &watcher.Options{
		ContainerAddr:            a.ContainerdAddr,
		MaxConcurrentReconcilers: a.MaxConcurrentReconcilers,
		MaxConcurrentPullers:     a.MaxConcurrentPullers,
		NodeName:                 nodeName,
	}); err != nil {
		log.Error(err, "unable to setup controllers")
		os.Exit(1)
	}

	if err = reporter.SetupWithManager(ctx, mgr, &reporter.Options{
		ContainerAddr:      a.ContainerdAddr,
		NodeName:           nodeName,
		Host:               a.Host,
		CertDir:            a.CertDir,
		CertName:           a.CertName,
		KeyName:            a.KeyName,
		InsecureSkipVerify: a.SkipInsecureVerify,
		TLSOpts: []func(*tls.Config){
			func(config *tls.Config) {
				config.InsecureSkipVerify = a.SkipInsecureVerify
			},
		},
	}); err != nil {
		log.Error(err, "unable to setup reporter")
		os.Exit(1)
	}

	return mgr.Start(ctx)
}
