package agent

import (
	"os"
	"time"

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

	return mgr.Start(ctx)
}
