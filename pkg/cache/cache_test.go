package cache

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"testing"
)

func TestCache_ResourcesFor(t *testing.T) {
	c := NewCache()

	obj := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pod1",
			Namespace: "ns1",
		},
	}

	c.CacheFor(obj)

	// what happens if we add multiple types in the same resource group.

}
