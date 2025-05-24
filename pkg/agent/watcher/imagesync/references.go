package imagesync

import (
	"sync"

	"ctx.sh/coral/pkg/store"
)

// References is a collection of images that are under management.  They are
// referenced by the image name and the digest of the image.
// References [name]->[[digest]] -> [count]].
type References struct {
	images map[Name]*store.Store[Digest]
	refs   map[Key]Digest

	sync.Mutex
}

func NewReferences() *References {
	return &References{
		images: make(map[Name]*store.Store[Digest]),
		refs:   make(map[Key]Digest),
	}
}

func (c *References) Add(uuid, name, digest string) {
	c.Lock()
	defer c.Unlock()

	// // We've already added the
	d, ok := c.addResourceRef(uuid, name, digest)
	if ok && d != digest {
		// The image has been updated, so we need to remove the old reference.
		c.images[name].Delete(d)
	} else if ok && d == digest {
		// We've already added this image to the collection, so no need to add it again.
		return
	}

	if _, ok := c.images[name]; !ok {
		c.images[name] = store.New[string]()
	}

	c.images[name].Add(digest)
}

func (c *References) Remove(uuid, name, digest string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.images[name]; !ok {
		return
	}

	c.images[name].Delete(digest)
	c.removeResourceRef(uuid, name)
}

func (c *References) IsReferenced(name string, digest string) bool {
	if _, ok := c.images[name]; !ok {
		return false
	}

	return c.images[name].IsReferenced(digest)
}

func (c *References) References(name, digest string) int {
	if _, ok := c.images[name]; !ok {
		return 0
	}

	return c.images[name].References(digest)
}

func (c *References) addResourceRef(uuid, name, digest string) (string, bool) {
	key := Key{
		UUID:  uuid,
		Image: name,
	}

	if v, ok := c.refs[key]; !ok {
		c.refs[key] = digest
		return "", false
	} else {
		return v, true
	}
}

func (c *References) removeResourceRef(uuid, name string) {
	key := Key{
		UUID:  uuid,
		Image: name,
	}

	delete(c.refs, key)
}
