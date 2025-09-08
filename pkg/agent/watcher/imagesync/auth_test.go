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

package imagesync

import (
	"testing"

	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
)

type AuthTestSuite struct {
	suite.Suite
}

func (s *AuthTestSuite) SetupTest() {}

func (s *AuthTestSuite) TearDownTest() {}

func TestAuthTestSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

func (s *AuthTestSuite) TestNewAuth() {
	// Test with no pull secrets
	_, err := NewAuth(nil)
	s.Require().NoError(err)

	// Test with a dockerconfigjson secret with fake credentials
	secret := corev1.Secret{
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			// Base64 encoded "fakeuser:fakepass"
			corev1.DockerConfigJsonKey: []byte(`{
					"auths": {
						"fake.registry.io": {
							"auth": "ZmFrZXVzZXI6ZmFrZXBhc3M="
						}
					}
				}`),
		},
	}
	auth, err := NewAuth([]corev1.Secret{secret})
	s.Require().NoError(err)
	s.Require().NotNil(auth)
}

func (s *AuthTestSuite) TestLookup() {
	// Fake secret with multiple registries.  The test secrets are base64 encoded
	// credentials for "fakeuser:fakepass" and "anotheruser:anotherpass".
	secret := corev1.Secret{
		Type: corev1.SecretTypeDockerConfigJson,
		Data: map[string][]byte{
			corev1.DockerConfigJsonKey: []byte(`{
					"auths": {
						"fake.registry.io": {
							"auth": "ZmFrZXVzZXI6ZmFrZXBhc3M="
						},
						"another.registry.io": {
							"auth": "YW5vdGhlcnVzZXI6YW5vdGhlcnBhc3M="
						}
					}
				}`,
			),
		},
	}
	auth, err := NewAuth([]corev1.Secret{secret})
	s.Require().NoError(err)

	// Lookup the auth for the registry
	result := auth.Lookup("fake.registry.io")
	s.Require().Equal("fakeuser", result[0].Username)
	s.Require().Equal("fakepass", result[0].Password)

	result = auth.Lookup("another.registry.io")
	s.Require().Equal("anotheruser", result[0].Username)
	s.Require().Equal("anotherpass", result[0].Password)
}
