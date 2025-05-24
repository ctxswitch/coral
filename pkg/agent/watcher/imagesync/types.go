package imagesync

type Digest = string

type Name = string

type Key struct {
	// The uuid of the imagesync resource.
	UUID string
	// The name of the image.
	Image string
}
