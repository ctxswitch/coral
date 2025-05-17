package image

import "errors"

type ImageError string

func (e ImageError) Error() string {
	return string(e)
}

const (
	ErrNotFound     ImageError = "not found"
	ErrNodeNotReady ImageError = "node is not ready"
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
