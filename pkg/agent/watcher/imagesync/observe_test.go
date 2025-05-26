package imagesync

import (
	"context"
	"ctx.sh/coral/pkg/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"
)

type ObserveTestSuite struct {
	client *mock.Client
	suite.Suite
}

func (s *ObserveTestSuite) SetupTest() {
	logger := zap.New(zap.UseDevMode(true), zap.Level(zapcore.Level(-8)))
	log.SetLogger(logger)

	s.client = mock.NewClient().
		WithLogger(logger).
		WithFixtureDirectory(filepath.Join("..", "..", "..", "..", "fixtures"))

	s.client.ApplyFixtureOrDie("nodes.yaml")
}

func (s *ObserveTestSuite) TearDownTest() {
	s.client.Reset()
}

func TestObserverTestSuite(t *testing.T) {
	suite.Run(t, new(ObserveTestSuite))
}

func (s *ObserveTestSuite) TestStateObserve_observe() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.client.ApplyFixtureOrDie(
		"example.yaml",
	)

	observed := NewObservedState()
	observer := &StateObserver{
		Client:   s.client,
		NodeName: "node1",
		Request: Request{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "example",
			},
		},
	}

	err := observer.observe(ctx, observed)

	s.NoError(err)
	s.NotNil(observed.ImageSync)
	s.Equal("example", observed.ImageSync.GetName())
	s.NotNil(observed.Node)
	s.Equal("node1", observed.Node.GetName())
	s.Empty(observed.PullSecrets)
}

func (s *ObserveTestSuite) TestStateObserve_observe_without_node() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.client.ApplyFixtureOrDie(
		"example.yaml",
	)

	observed := NewObservedState()
	observer := &StateObserver{
		Client:   s.client,
		NodeName: "notfound",
		Request: Request{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "example",
			},
		},
	}

	err := observer.observe(ctx, observed)

	s.Error(err)
	s.ErrorIs(err, ErrNodeNotFound)
	s.Nil(observed.ImageSync)
	s.Nil(observed.Node)
	s.Empty(observed.PullSecrets)
}

func (s *ObserveTestSuite) TestStateObserve_observe_node_not_matched() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.client.ApplyFixtureOrDie(
		"example_with_selector.yaml",
	)

	observed := NewObservedState()
	observer := &StateObserver{
		Client:   s.client,
		NodeName: "node1", // This node does not match the ImageSync's node selector
		Request: Request{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "example",
			},
		},
	}

	err := observer.observe(ctx, observed)

	s.Error(err)
	s.ErrorIs(err, ErrNodeMatch)
	s.NotNil(observed.Node)
	s.NotNil(observed.ImageSync)
	s.Empty(observed.PullSecrets)
}

func (s *ObserveTestSuite) TestStateObserve_observe_node_matched() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.client.ApplyFixtureOrDie(
		"example_with_selector.yaml",
	)

	var node corev1.Node
	err := s.client.Get(ctx, client.ObjectKey{Name: "node1"}, &node)
	s.NoError(err)

	node.SetLabels(map[string]string{
		"role": "app",
	})
	err = s.client.Update(ctx, &node)
	s.NoError(err)

	observed := NewObservedState()
	observer := &StateObserver{
		Client:   s.client,
		NodeName: "node1", // This node does not match the ImageSync's node selector
		Request: Request{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "example",
			},
		},
	}

	err = observer.observe(ctx, observed)

	s.NoError(err)
	s.NotNil(observed.Node)
	s.NotNil(observed.ImageSync)
	s.Empty(observed.PullSecrets)
}

func (s *ObserveTestSuite) TestStateObserve_observe_without_imagesync() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	observed := NewObservedState()
	observer := &StateObserver{
		Client:   s.client,
		NodeName: "node1",
		Request: Request{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "notfound",
			},
		},
	}

	err := observer.observe(ctx, observed)

	s.Error(err)
	s.ErrorIs(err, ErrImageSyncNotFound)
}

func (s *ObserveTestSuite) TestStateObserve_observe_with_pullsecrets() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.client.ApplyFixtureOrDie(
		"example.yaml",
	)

	observed := NewObservedState()
	observer := &StateObserver{
		Client:   s.client,
		NodeName: "node1",
		Request: Request{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "example-with-secrets",
			},
		},
	}

	err := observer.observe(ctx, observed)

	s.NoError(err)
	s.NotNil(observed.ImageSync)
	s.Equal("example-with-secrets", observed.ImageSync.GetName())
	s.NotNil(observed.Node)
	s.Equal("node1", observed.Node.GetName())
	s.NotEmpty(observed.PullSecrets)
	s.Len(observed.PullSecrets, 1)
	s.Equal("fake-credentials", observed.PullSecrets[0].GetName())
}

func (s *ObserveTestSuite) TestStateObserve_observe_with_missing_pullsecrets() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.client.ApplyFixtureOrDie(
		"example.yaml",
	)

	observed := NewObservedState()
	observer := &StateObserver{
		Client:   s.client,
		NodeName: "node1",
		Request: Request{
			NamespacedName: types.NamespacedName{
				Namespace: "default",
				Name:      "example-with-missing-secrets",
			},
		},
	}

	err := observer.observe(ctx, observed)

	s.Error(err)
	s.ErrorIs(err, ErrPullSecretsNotFound)
	s.NotNil(observed.ImageSync)
	s.Equal("example-with-missing-secrets", observed.ImageSync.GetName())
	s.NotNil(observed.Node)
	s.Equal("node1", observed.Node.GetName())
}

func (s *ObserveTestSuite) TestStateObserve_observe_node_not_ready() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	s.client.ApplyFixtureOrDie(
		"example.yaml",
	)

	tests := []struct {
		nodeName       string
		memoryPressure string
		diskPressure   string
		pidPressure    string
		ready          string
		err            error
	}{
		{"node1", "False", "False", "False", "True", nil},
		{"node1", "False", "False", "False", "False", ErrNodeNotReady},
		{"node1", "True", "False", "False", "False", ErrNodeNotReady},
		{"node1", "False", "True", "False", "False", ErrNodeNotReady},
		{"node1", "False", "False", "True", "False", ErrNodeNotReady},
	}

	for _, tt := range tests {
		var node corev1.Node
		err := s.client.Get(ctx, client.ObjectKey{Name: tt.nodeName}, &node)
		s.NoError(err)

		node.Status.Conditions = []corev1.NodeCondition{
			{
				Type:   corev1.NodeReady,
				Status: corev1.ConditionStatus(tt.ready),
			},
			{
				Type:   corev1.NodeMemoryPressure,
				Status: corev1.ConditionStatus(tt.memoryPressure),
			},
			{
				Type:   corev1.NodeDiskPressure,
				Status: corev1.ConditionStatus(tt.diskPressure),
			},
			{
				Type:   corev1.NodePIDPressure,
				Status: corev1.ConditionStatus(tt.pidPressure),
			},
		}

		err = s.client.Status().Update(ctx, &node)
		s.NoError(err)

		observed := NewObservedState()
		observer := &StateObserver{
			Client:   s.client,
			NodeName: tt.nodeName,
			Request: Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "example",
				},
			},
		}

		err = observer.observe(ctx, observed)

		if tt.err != nil {
			s.Error(err)
			s.ErrorIs(err, tt.err)
		} else {
			s.NoError(err)
		}
	}
}
