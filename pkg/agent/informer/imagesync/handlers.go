package imagesync

import (
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
)

type Handler struct{}

func (h *Handler) OnAdd(obj interface{}, init bool) {
	isync, ok := obj.(*coralv1beta1.ImageSync)
	if !ok {
		return
	}

	h.add(isync)
}

func (h *Handler) OnUpdate(oldObj, newObj interface{}) {
	isync, ok := newObj.(*coralv1beta1.ImageSync)
	if !ok {
		return
	}

	h.add(isync)
}

func (h *Handler) OnDelete(obj interface{}) {
	isync, ok := obj.(*coralv1beta1.ImageSync)
	if !ok {
		return
	}

	h.delete(isync)
}

func (h *Handler) add(obj *coralv1beta1.ImageSync) {

}

func (h *Handler) delete(obj *coralv1beta1.ImageSync) {

}
