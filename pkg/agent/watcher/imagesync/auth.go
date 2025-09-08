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
	corev1 "k8s.io/api/core/v1"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
	"k8s.io/kubernetes/pkg/credentialprovider"
	"k8s.io/kubernetes/pkg/credentialprovider/secrets"
)

type Auth struct {
	keyring credentialprovider.DockerKeyring
}

func NewAuth(pullSecrets []corev1.Secret) (*Auth, error) {
	defaultKeyring := credentialprovider.NewDefaultDockerKeyring()
	keyring, err := secrets.MakeDockerKeyring(pullSecrets, defaultKeyring)
	if err != nil {
		return nil, err
	}

	return &Auth{
		keyring: keyring,
	}, nil
}

func (a *Auth) Lookup(name string) []*runtime.AuthConfig {
	auth := a.authLookup(name)
	runtimeAuth := make([]*runtime.AuthConfig, len(auth))
	for i, v := range auth {
		runtimeAuth[i] = &runtime.AuthConfig{
			Username:      v.Username,
			Password:      v.Password,
			Auth:          v.Auth,
			ServerAddress: v.ServerAddress,
			IdentityToken: v.IdentityToken,
			RegistryToken: v.RegistryToken,
		}
	}

	return runtimeAuth
}

func (a *Auth) authLookup(name string) []credentialprovider.TrackedAuthConfig {
	auth, found := a.keyring.Lookup(name)
	if !found {
		return []credentialprovider.TrackedAuthConfig{}
	}

	return auth
}
