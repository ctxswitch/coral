package image

import (
	"context"

	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
)

type ImageClient interface {
	// Pull pulls a container image if not present.
	Pull(ctx context.Context, id, name, ref string) error
	// Delete deletes a container image that is not in use.
	Delete(ctx context.Context, id, name, ref string) error
	// Matches returns true if the node matches the selectors.
	Matches(ctx context.Context, selectors []coralv1beta1.NodeSelector) (bool, error)
}

type Info struct {
	// ID is the image ID.
	ID string
	// Name is the name of the image.
	Name string
	// Tags are the tags associated with the image.
	Tags []string
	// References are the number of imagesync resources that reference this image.
	References int
}

type Digest = string

type Name = string

type Key struct {
	// The uuid of the imagesync resource.
	UUID string
	// The name of the image.
	Image string
}
