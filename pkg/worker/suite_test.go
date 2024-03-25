package worker

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	stvziov1 "stvz.io/coral/pkg/apis/stvz.io/v1"
	"stvz.io/coral/pkg/util"
)

var (
	cfg    *rest.Config
	cli    client.Client
	env    *envtest.Environment
	ctx    context.Context
	cancel context.CancelFunc
)

func TestImageController(t *testing.T) {
	RegisterFailHandler(Fail)
	suiteConfig, _ := GinkgoConfiguration()
	suiteConfig.ParallelTotal = 1
	RunSpecs(t, "Image Controller Suite", suiteConfig)
}

var _ = BeforeSuite(func() {
	logf.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))

	ctx, cancel = context.WithCancel(context.Background())

	By("bootstrapping test environment")
	env = &envtest.Environment{
		CRDDirectoryPaths:     []string{filepath.Join("..", "..", "config", "crd")},
		ErrorIfCRDPathMissing: true,
		BinaryAssetsDirectory: filepath.Join("..", "..", "bin", "k8s",
			fmt.Sprintf("1.29.1-%s-%s", runtime.GOOS, runtime.GOARCH)),
	}

	var err error
	cfg, err = env.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = stvziov1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = corev1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())
	err = appsv1.AddToScheme(scheme.Scheme)
	Expect(err).NotTo(HaveOccurred())

	// +kubebuilder:scaffold:scheme

	cli, err = client.New(cfg, client.Options{Scheme: scheme.Scheme})
	Expect(err).ToNot(HaveOccurred())
	Expect(cli).ToNot(BeNil())
})

var _ = AfterSuite(func() {
	cancel()
	By("tearing down the test environment")
	err := env.Stop()
	Expect(err).NotTo(HaveOccurred())
})

func SetupTest(ctx context.Context) *corev1.Namespace {
	ns := &corev1.Namespace{}

	BeforeEach(func() {
		*ns = util.GenerateTestNamespace()

		err := cli.Create(ctx, ns)
		Expect(err).NotTo(HaveOccurred(), "failed to create test namespace")

		go func() {
			defer GinkgoRecover()
			Expect(err).ToNot(HaveOccurred(), "failed to run manager")
		}()
	})

	AfterEach(func() {
		gexec.KillAndWait(5 * time.Second)
		err := cli.Delete(ctx, ns)
		Expect(err).NotTo(HaveOccurred(), "failed to delete test namespace")
	})

	logf.Log.Info("namespace created", "name", ns.Name)
	return ns
}

func ensureImages(ctx context.Context, imgs ...*stvziov1.Image) {
	for _, img := range imgs {
		By(fmt.Sprintf("creating image: %s", img.Name))
		Expect(cli.Create(ctx, img)).To(Succeed())
		Eventually(func() bool {
			err := cli.Get(ctx, client.ObjectKey{Name: img.Name, Namespace: img.Namespace}, img)
			return err == nil
		}, timeout, interval).Should(BeTrue())
	}
}
