package cache

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Cache struct {
	Resources map[schema.GroupVersionKind]*Resources
	client.Client
}

func NewCache() *Cache {
	return &Cache{
		Resources: make(map[schema.GroupVersionKind]*Resources),
	}
}

func (c *Cache) CacheFor(obj client.Object) *Resources {
	gvk := obj.GetObjectKind().GroupVersionKind()

	// TODO: Should we only allow certain GVKs to be cached?
	if _, ok := c.Resources[gvk]; !ok {
		c.Resources[gvk] = NewResources(gvk)
	}

	return c.Resources[gvk]
}
