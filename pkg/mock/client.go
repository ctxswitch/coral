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

package mock

import (
	"os"
	"path/filepath"
	"strings"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/testing"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type Client struct {
	fixtureDir string
	log        logr.Logger
	tracker    testing.ObjectTracker
	scheme     *runtime.Scheme
	client.Client
}

// NewClient returns a mock (fake) client for testing. The fixtures are
// not automatically loaded into the cache.  Individual fixtures can be loaded
// using the ApplyFixtureOrDie method.
func NewClient() *Client {
	s := scheme.Scheme
	_ = coralv1beta1.AddToScheme(s)

	tracker := testing.NewObjectTracker(s, scheme.Codecs.UniversalDecoder())
	c := fake.NewClientBuilder().
		WithObjectTracker(tracker).
		WithScheme(s).
		WithStatusSubresource(&coralv1beta1.ImageSync{}).
		Build()

	return &Client{
		fixtureDir: "",
		log:        logr.Discard(),
		scheme:     s,
		tracker:    tracker,
		Client:     c,
	}
}

// WithLogger sets the logger for the client.
func (m *Client) WithLogger(log logr.Logger) *Client {
	m.log = log
	return m
}

func (m *Client) WithFixtureDirectory(dir string) *Client {
	m.fixtureDir = dir
	return m
}

// ApplyFixtureOrDie loads a single fixture into the cache.  The fixture must be in a
// recognizable format for the universal deserializer.
func (m *Client) ApplyFixtureOrDie(path ...string) {
	decoder := scheme.Codecs.UniversalDeserializer()

	file := filepath.Join(path...)

	f := filepath.Join(m.fixtureDir, file)
	data, err := os.ReadFile(f)
	if err != nil {
		panic(err)
	}

	sections := strings.Split(string(data), "---")

	for _, section := range sections {
		data = []byte(section)
		decoded, _, err := decoder.Decode(data, nil, nil)
		if err != nil {
			panic(err)
		}

		obj, has := decoded.(client.Object)
		if !has {
			panic("decoded object is not a client.Object")
		}

		// Fake some of the creation metadata.  There's probably a few other
		// things that could be useful.
		if obj.GetCreationTimestamp().Time.IsZero() {
			obj.SetCreationTimestamp(metav1.Time{
				Time: metav1.Now().Time,
			})
		}

		// TODO: For some reason the client does not see the VersionKind in the scheme. So
		//   we panic here.
		// isNamespaced, err := m.Client.IsObjectNamespaced(obj)
		// if err != nil {
		// 	panic(fmt.Sprintf("error checking if object is namespaced: %v", err))
		// }
		//
		// if isNamespaced && obj.GetNamespace() == "" {
		// 	// If the object is namespaced, but the namespace is not set, we
		// 	// set it to the default namespace.
		// 	obj.SetNamespace(corev1.NamespaceDefault)
		// }

		err = m.tracker.Add(obj)
		if err != nil {
			panic(err)
		}
	}
}

func (m *Client) Reset() {
	m.tracker = testing.NewObjectTracker(m.scheme, scheme.Codecs.UniversalDecoder())
}

var _ client.Client = &Client{}
