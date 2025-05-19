package cache

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var (
	podGVK  = corev1.SchemeGroupVersion.WithKind("Pod")
	podType = metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}
	configMapType = metav1.TypeMeta{
		Kind:       "ConfigMap",
		APIVersion: "v1",
	}
)

func TestResources_Add(t *testing.T) {
	res := NewResources(podGVK)

	obj1 := &corev1.Pod{
		TypeMeta: podType,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
		},
	}

	nn := types.NamespacedName{
		Name:      obj1.GetName(),
		Namespace: obj1.GetNamespace(),
	}
	err := res.Add(obj1)

	assert.NoError(t, err)
	assert.Equal(t, 1, len(res.items))
	assert.Equal(t, obj1, res.items[nn])

	obj2 := &corev1.ConfigMap{
		TypeMeta: configMapType,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
		},
	}
	err = res.Add(obj2)
	assert.Error(t, err)
	assert.Equal(t, 1, len(res.items))
}

func TestResources_Get(t *testing.T) {
	res := NewResources(podGVK)

	obj1 := &corev1.Pod{
		TypeMeta: podType,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
		},
	}

	nn := types.NamespacedName{
		Name:      obj1.GetName(),
		Namespace: obj1.GetNamespace(),
	}
	res.items[nn] = obj1

	obj, found := res.Get(nn)
	assert.Equal(t, 1, len(res.items))
	assert.True(t, found)
	assert.Equal(t, obj1, obj)
}

func TestResources_Remove(t *testing.T) {
	res := NewResources(podGVK)

	obj1 := &corev1.Pod{
		TypeMeta: podType,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
		},
	}

	obj2 := &corev1.Pod{
		TypeMeta: podType,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "ns2",
		},
	}

	nn1 := types.NamespacedName{
		Name:      obj1.GetName(),
		Namespace: obj1.GetNamespace(),
	}

	nn2 := types.NamespacedName{
		Name:      obj2.GetName(),
		Namespace: obj2.GetNamespace(),
	}

	res.items[nn1] = obj1
	res.items[nn2] = obj2

	assert.Equal(t, 2, len(res.items))

	res.Remove(obj1)
	assert.Equal(t, 1, len(res.items))
	assert.NotContains(t, res.items, nn1)
	assert.Contains(t, res.items, nn2)
}

func TestResources_CreateOrUpdate(t *testing.T) {
	res := NewResources(podGVK)

	obj1 := &corev1.Pod{
		TypeMeta: podType,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
		},
	}

	obj2 := &corev1.Pod{
		TypeMeta: podType,
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod2",
			Namespace: "ns2",
		},
	}

	nn1 := types.NamespacedName{
		Name:      obj1.GetName(),
		Namespace: obj1.GetNamespace(),
	}

	nn2 := types.NamespacedName{
		Name:      obj2.GetName(),
		Namespace: obj2.GetNamespace(),
	}

	res.items[nn1] = obj1

	assert.Contains(t, res.items, nn1)
	assert.NotContains(t, res.items, nn2)

	err := res.CreateOrUpdate(obj2)
	assert.NoError(t, err)
	assert.Contains(t, res.items, nn2)
	assert.Equal(t, obj2, res.items[nn2])

	obj1.SetNamespace("updated")
	err = res.CreateOrUpdate(obj1)
	assert.NoError(t, err)
	assert.Contains(t, res.items, nn2)
	assert.Equal(t, "updated", res.items[nn1].GetNamespace())
}
