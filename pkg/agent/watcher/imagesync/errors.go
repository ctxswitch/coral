package imagesync

type WatchError string

func (e WatchError) Error() string {
	return string(e)
}

const (
	ErrNodeNotFound      WatchError = "node not found"
	ErrNodeNotReady      WatchError = "node not ready"
	ErrNodeMatch         WatchError = "node does not match match"
	ErrImageSyncNotFound WatchError = "imagesync notfound"
)
