package imagesync

import (
	"context"
	"ctx.sh/coral/pkg/agent/image"
	"ctx.sh/coral/pkg/mock"
	"ctx.sh/coral/pkg/queue"
	smock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
)

type WatcherTestSuite struct {
	client *mock.Client
	suite.Suite
}

func (s *WatcherTestSuite) SetupTest() {
	logger := zap.New(zap.UseDevMode(true), zap.Level(zapcore.Level(-8)))
	log.SetLogger(logger)

	s.client = mock.NewClient().
		WithLogger(logger).
		WithFixtureDirectory(filepath.Join("..", "..", "..", "..", "fixtures"))

	s.client.ApplyFixtureOrDie("nodes.yaml")
}

func (s *WatcherTestSuite) TearDownTest() {
	s.client.Reset()
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}

func (s *WatcherTestSuite) TestReconcile_pull() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.client.ApplyFixtureOrDie("imagesync-agent-watcher.yaml")

	wq := queue.New(1)
	imageClient := mock.NewMockImageClient(s.T())
	imageClient.EXPECT().Pull(smock.Anything, "b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/golang:latest", smock.Anything).Return(image.Info{
		ID:   "golang-id",
		Name: "docker.io/library/golang:latest",
		Tags: []string{"latest"},
	}, nil).Once()
	imageClient.EXPECT().Pull(smock.Anything, "b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/nginx:latest", smock.Anything).Return(image.Info{
		ID:   "nginx-id",
		Name: "docker.io/library/nginx:latest",
		Tags: []string{"latest"},
	}, nil).Once()

	watcher := &Watcher{
		workQueue:   wq,
		nodeName:    "node1",
		imageClient: imageClient,
		collection:  NewReferences(),
		Client:      s.client,
	}

	result, err := watcher.Reconcile(ctx, Request{
		NamespacedName: types.NamespacedName{
			Name:      "example",
			Namespace: "default",
		},
	})

	s.NoError(err, "Reconcile should not return an error")
	s.Empty(ctrl.Result{}, result)

	var node corev1.Node
	err = s.client.Get(ctx, types.NamespacedName{Name: "node1"}, &node)
	s.NoError(err, "Should be able to get node")
	s.Contains(node.GetLabels(), "imagesync.coral.ctx.sh/7f88bbf3e814bc26c05c719c470303da")
	s.Equal("present", node.GetLabels()["imagesync.coral.ctx.sh/7f88bbf3e814bc26c05c719c470303da"])
	s.Contains(node.GetLabels(), "imagesync.coral.ctx.sh/819de3aee13d29b3d12a28494ac35519")
	s.Equal("present", node.GetLabels()["imagesync.coral.ctx.sh/819de3aee13d29b3d12a28494ac35519"])
}

func (s *WatcherTestSuite) TestReconcile_delete() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set the labels on the node to simulate an existing successful reconcile.
	var node corev1.Node
	err := s.client.Get(ctx, types.NamespacedName{Name: "node1"}, &node)
	s.NoError(err)
	node.SetLabels(map[string]string{
		"imagesync.coral.ctx.sh/7f88bbf3e814bc26c05c719c470303da": "present",
		"imagesync.coral.ctx.sh/819de3aee13d29b3d12a28494ac35519": "present",
	})
	err = s.client.Update(ctx, &node)
	s.NoError(err)

	// Apply the fixture that simulates the deletion of the ImageSync resource and run the reconcile.
	s.client.ApplyFixtureOrDie("imagesync-agent-watcher-deleted.yaml")

	wq := queue.New(1)
	imageClient := mock.NewMockImageClient(s.T())
	imageClient.EXPECT().Delete(smock.Anything, "b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/golang:latest").Return(image.Info{
		ID:   "golang-id",
		Name: "docker.io/library/golang:latest",
		Tags: []string{"latest"},
	}, nil).Once()
	imageClient.EXPECT().Delete(smock.Anything, "b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/nginx:latest").Return(image.Info{
		ID:   "nginx-id",
		Name: "docker.io/library/nginx:latest",
		Tags: []string{"latest"},
	}, nil).Once()

	watcher := &Watcher{
		workQueue:   wq,
		nodeName:    "node1",
		imageClient: imageClient,
		collection:  NewReferences(),
		Client:      s.client,
	}

	watcher.collection.Add("b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/golang:latest", "golang-id")
	watcher.collection.Add("b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/nginx:latest", "nginx-id")

	result, err := watcher.Reconcile(ctx, Request{
		NamespacedName: types.NamespacedName{
			Name:      "example",
			Namespace: "default",
		},
	})

	s.NoError(err)
	s.Empty(ctrl.Result{}, result)

	var updatedNode corev1.Node
	err = s.client.Get(ctx, types.NamespacedName{Name: "node1"}, &updatedNode)
	s.NoError(err)
	s.NotContains(updatedNode.GetLabels(), "imagesync.coral.ctx.sh/7f88bbf3e814bc26c05c719c470303da")
	s.NotContains(updatedNode.GetLabels(), "imagesync.coral.ctx.sh/819de3aee13d29b3d12a28494ac35519")
}

func (s *WatcherTestSuite) TestReconcile_delete_single() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set the labels on the node to simulate an existing successful reconcile.
	var node corev1.Node
	err := s.client.Get(ctx, types.NamespacedName{Name: "node1"}, &node)
	s.NoError(err)
	node.SetLabels(map[string]string{
		"imagesync.coral.ctx.sh/7f88bbf3e814bc26c05c719c470303da": "present",
		"imagesync.coral.ctx.sh/819de3aee13d29b3d12a28494ac35519": "present",
	})
	err = s.client.Update(ctx, &node)
	s.NoError(err)

	// Apply the fixture that simulates the deletion of the ImageSync resource and run the reconcile.
	s.client.ApplyFixtureOrDie("imagesync-agent-watcher-deleted-single.yaml")

	wq := queue.New(1)
	imageClient := mock.NewMockImageClient(s.T())
	imageClient.EXPECT().Pull(smock.Anything, "b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/golang:latest", smock.Anything).Return(image.Info{
		ID:   "golang-id",
		Name: "docker.io/library/golang:latest",
		Tags: []string{"latest"},
	}, nil).Once()
	imageClient.EXPECT().Delete(smock.Anything, "b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/nginx:latest").Return(image.Info{
		ID:   "nginx-id",
		Name: "docker.io/library/nginx:latest",
		Tags: []string{"latest"},
	}, nil).Once()

	watcher := &Watcher{
		workQueue:   wq,
		nodeName:    "node1",
		imageClient: imageClient,
		collection:  NewReferences(),
		Client:      s.client,
	}

	watcher.collection.Add("b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/golang:latest", "golang-id")
	watcher.collection.Add("b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/nginx:latest", "nginx-id")

	result, err := watcher.Reconcile(ctx, Request{
		NamespacedName: types.NamespacedName{
			Name:      "example",
			Namespace: "default",
		},
	})

	s.NoError(err)
	s.Empty(ctrl.Result{}, result)

	var updatedNode corev1.Node
	err = s.client.Get(ctx, types.NamespacedName{Name: "node1"}, &updatedNode)
	s.NoError(err)
	s.Contains(updatedNode.GetLabels(), "imagesync.coral.ctx.sh/7f88bbf3e814bc26c05c719c470303da")
	s.NotContains(updatedNode.GetLabels(), "imagesync.coral.ctx.sh/819de3aee13d29b3d12a28494ac35519")
}

func (s *WatcherTestSuite) TestReconcile_node_selectors_changed() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set the labels on the node to simulate an existing successful reconcile.
	var node corev1.Node
	err := s.client.Get(ctx, types.NamespacedName{Name: "node1"}, &node)
	s.NoError(err)
	node.SetLabels(map[string]string{
		"imagesync.coral.ctx.sh/7f88bbf3e814bc26c05c719c470303da": "present",
		"imagesync.coral.ctx.sh/819de3aee13d29b3d12a28494ac35519": "present",
	})
	err = s.client.Update(ctx, &node)
	s.NoError(err)

	s.client.ApplyFixtureOrDie("imagesync-agent-watcher-node-selector.yaml")

	wq := queue.New(1)
	imageClient := mock.NewMockImageClient(s.T())
	imageClient.EXPECT().Delete(smock.Anything, "b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/golang:latest").Return(image.Info{
		ID:   "golang-id",
		Name: "docker.io/library/golang:latest",
		Tags: []string{"latest"},
	}, nil).Once()
	imageClient.EXPECT().Delete(smock.Anything, "b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/nginx:latest").Return(image.Info{
		ID:   "nginx-id",
		Name: "docker.io/library/nginx:latest",
		Tags: []string{"latest"},
	}, nil).Once()

	watcher := &Watcher{
		workQueue:   wq,
		nodeName:    "node1",
		imageClient: imageClient,
		collection:  NewReferences(),
		Client:      s.client,
	}

	watcher.collection.Add("b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/golang:latest", "golang-id")
	watcher.collection.Add("b7f01748-4d55-4bc3-939a-a458c19ca533", "docker.io/library/nginx:latest", "nginx-id")

	result, err := watcher.Reconcile(ctx, Request{
		NamespacedName: types.NamespacedName{
			Name:      "example",
			Namespace: "default",
		},
	})

	s.NoError(err)
	s.Empty(ctrl.Result{}, result)

	var updatedNode corev1.Node
	err = s.client.Get(ctx, types.NamespacedName{Name: "node1"}, &updatedNode)
	s.NoError(err)
	s.NotContains(updatedNode.GetLabels(), "imagesync.coral.ctx.sh/7f88bbf3e814bc26c05c719c470303da")
	s.NotContains(updatedNode.GetLabels(), "imagesync.coral.ctx.sh/819de3aee13d29b3d12a28494ac35519")
}
