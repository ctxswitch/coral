package cache

import (
	"fmt"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
)

type Resources struct {
	size  uint64
	items map[types.NamespacedName]client.Object // V may need to be something different to account for resources...  maybe a struct with compare/get func and object.
	gvk   schema.GroupVersionKind
	sync.Mutex
}

func NewResources(gvk schema.GroupVersionKind) *Resources {
	return &Resources{
		size:  0,
		items: make(map[types.NamespacedName]client.Object),
		gvk:   gvk,
	}
}

func (r *Resources) Add(v client.Object) error {
	r.Lock()
	defer r.Unlock()

	gvk := v.GetObjectKind().GroupVersionKind()
	if gvk != r.gvk {
		return fmt.Errorf("expected GVK %s, got %s", r.gvk.String(), gvk.String())
	}

	nn := types.NamespacedName{
		Name:      v.GetName(),
		Namespace: v.GetNamespace(),
	}

	if _, ok := r.items[nn]; ok {
		return nil
	}

	r.items[nn] = v
	r.size++

	return nil
}

func (r *Resources) Remove(v client.Object) {
	r.Lock()
	defer r.Unlock()

	nn := types.NamespacedName{
		Name:      v.GetName(),
		Namespace: v.GetNamespace(),
	}

	if _, ok := r.items[nn]; !ok {
		return
	}

	delete(r.items, nn)
	r.size--
}

func (r *Resources) CreateOrUpdate(v client.Object) error {
	r.Remove(v)
	return r.Add(v)
}

func (r *Resources) Get(k types.NamespacedName) (client.Object, bool) {
	r.Lock()
	defer r.Unlock()

	v, ok := r.items[k]
	
	return v, ok
}

func (r *Resources) Size() int {
	r.Lock()
	defer r.Unlock()

	return int(r.size)
}
