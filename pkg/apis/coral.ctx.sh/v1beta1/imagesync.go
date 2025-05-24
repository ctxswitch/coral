package v1beta1

import (
	"crypto/md5" // #nosec G501
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (i *ImageSync) GetRevisionHash() string {
	hasher := md5.New() // #nosec G401
	obj := ImageSync{
		ObjectMeta: metav1.ObjectMeta{
			Name:      i.GetName(),
			Namespace: i.GetNamespace(),
			UID:       i.GetUID(),
		},
		Spec: ImageSyncSpec{
			Images: i.Spec.Images,
		},
	}
	DeepHashObject(hasher, obj)
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func (i *ImageSync) HasChanged() bool {
	return i.Status.Revision != i.GetRevisionHash()
}

func (i *ImageSync) HasNotChanged() bool {
	return i.Status.Revision == i.GetRevisionHash()
}
