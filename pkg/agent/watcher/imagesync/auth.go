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
	defaultKeyring := credentialprovider.NewDockerKeyring()
	keyring, err := secrets.MakeDockerKeyring(pullSecrets, defaultKeyring)
	if err != nil {
		return nil, err
	}

	return &Auth{
		keyring: keyring,
	}, nil
}

func (a *Auth) Lookup(name string) []*runtime.AuthConfig {
	// TODO: should probably cache this, but for now, it's not super expensive.
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

func (a *Auth) authLookup(name string) []credentialprovider.AuthConfig {
	auth, found := a.keyring.Lookup(name)
	if !found {
		return []credentialprovider.AuthConfig{}
	}

	return auth
}
