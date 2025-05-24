package image

import (
	"context"
	runtime "k8s.io/cri-api/pkg/apis/runtime/v1"
)

type ImageClient interface {
	// Pull pulls a container image if not present.
	Pull(ctx context.Context, uid, name string, auth []*runtime.AuthConfig) (Info, error)
	// Delete deletes a container image that is not in use.
	Delete(ctx context.Context, uid, name string) (Info, error)
	// Status returns the status of an image.
	Status(ctx context.Context, name string) (Info, error)
}

type Info struct {
	// ID is the image ID.
	ID string
	// Name is the name of the image.
	Name string
	// Tags are the tags associated with the image.
	Tags []string
}
