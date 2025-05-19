package v1beta1

import (
	"fmt"
	"hash"

	"k8s.io/apimachinery/pkg/util/dump"
)

// DeepHashObject computes a hash of the given object using the
// dump utility which dereferences any pointers to return the actual
// values and not the addresses.
func DeepHashObject(hasher hash.Hash, objectToWrite interface{}) {
	hasher.Reset()
	_, _ = fmt.Fprintf(hasher, "%v", dump.ForHash(objectToWrite))
}
