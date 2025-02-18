package agent

import (
	"os"
	"time"

	"ctx.sh/coral/pkg/agent"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"github.com/go-logr/logr"
	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	crun "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/cri-client/pkg/util"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/manager/signals"
)

const (
	ConnectionTimeout  time.Duration = 30 * time.Second
	MaxCallRecvMsgSize int           = 1024 * 1024 * 32
)

type Agent struct {
	ContainerdAddr string
	LogLevel       int8
	Workers        int
}

func (a *Agent) RunE(cmd *cobra.Command, args []string) error {
	log := zap.New(
		zap.Level(zapcore.Level(a.LogLevel) * -1),
	)

	ctx := signals.SetupSignalHandler()
	ctx = logr.NewContext(ctx, log)

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Error(nil, "NODE_NAME must be set in the environment.")
		os.Exit(1)
	}

	ims, rts, err := a.connectContainerRuntime(a.ContainerdAddr)
	if err != nil {
		log.Error(err, "failed to connect to container runtime")
		os.Exit(1)
	}

	c, err := a.connectKubeClient()
	if err != nil {
		log.Error(err, "failed to connect to kube client")
		os.Exit(1)
	}

	ag := agent.NewAgent(&agent.Options{
		ImageServiceClient:   ims,
		RuntimeServiceClient: rts,
		Client:               c,
	})

	if err := ag.Start(ctx); err != nil {
		log.Error(err, "failed to start agent")
		os.Exit(1)
	}

	return nil
}

func (a *Agent) connectContainerRuntime(addr string) (crun.ImageServiceClient, crun.RuntimeServiceClient, error) {
	addr, dialer, err := util.GetAddressAndDialer(addr)
	if err != nil {
		klog.ErrorS(err, "Get container runtime address failed")
		return nil, nil, err
	}

	conn, err := grpc.NewClient(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithContextDialer(dialer),
		grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(MaxCallRecvMsgSize)),
	)
	if err != nil {
		klog.ErrorS(err, "Connect remote image service failed", "address", addr)
		return nil, nil, err
	}

	ims := crun.NewImageServiceClient(conn)
	rts := crun.NewRuntimeServiceClient(conn)
	return ims, rts, nil
}

func (a *Agent) connectKubeClient() (client.Client, error) {
	scheme := runtime.NewScheme()
	_ = coralv1beta1.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)

	c, err := client.New(config.GetConfigOrDie(), client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, err
	}

	return c, nil
}
