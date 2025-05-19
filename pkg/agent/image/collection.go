package image

import (
	"sync"

	"ctx.sh/coral/pkg/agent/store"
)

// Collection is a collection of images that are under management.  They are
// referenced by the image name and the digest of the image.
// Collection [name]->[[digest]] -> [count]].
type Collection struct {
	images map[Name]*store.Store[Digest]
	refs   map[Key]Digest

	sync.Mutex
}

func NewCollection() *Collection {
	return &Collection{
		images: make(map[Name]*store.Store[Digest]),
		refs:   make(map[Key]Digest),
	}
}

func (c *Collection) Add(uuid, name, digest string) {
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

func (c *Collection) Remove(uuid, name, digest string) {
	c.Lock()
	defer c.Unlock()

	if _, ok := c.images[name]; !ok {
		return
	}

	c.images[name].Delete(digest)
	c.removeResourceRef(uuid, name)
}

func (c *Collection) IsReferenced(name string, digest string) bool {
	if _, ok := c.images[name]; !ok {
		return false
	}

	return c.images[name].IsReferenced(digest)
}

func (c *Collection) References(name, digest string) int {
	if _, ok := c.images[name]; !ok {
		return 0
	}

	return c.images[name].References(digest)
}

func (c *Collection) addResourceRef(uuid, name, digest string) (string, bool) {
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

func (c *Collection) removeResourceRef(uuid, name string) {
	key := Key{
		UUID:  uuid,
		Image: name,
	}

	delete(c.refs, key)
}
