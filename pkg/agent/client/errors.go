package client

import "errors"

type ImageError string

func (e ImageError) Error() string {
	return string(e)
}

const (
	ErrNotFound     ImageError = "not found"
	ErrUnauthorized ImageError = "unauthorized"
)

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}
