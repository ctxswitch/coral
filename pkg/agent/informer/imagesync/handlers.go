package imagesync

import (
	"ctx.sh/coral/pkg/agent/event"
	coralv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"fmt"
	"github.com/go-logr/logr"
	"reflect"
)

type Handler struct {
	logger logr.Logger
	events chan<- event.Event
}

func (h *Handler) OnAdd(obj interface{}, init bool) {
	isync, ok := obj.(*coralv1beta1.ImageSync)
	if !ok {
		h.logger.Error(fmt.Errorf("unexpected type"), "type", reflect.TypeOf(obj))
		return
	}

	h.add(isync)
}

func (h *Handler) OnUpdate(oldObj, newObj interface{}) {
	isync, ok := newObj.(*coralv1beta1.ImageSync)
	if !ok {
		h.logger.Error(fmt.Errorf("unexpected type"), "type", reflect.TypeOf(newObj))
		return
	}

	h.add(isync)
}

func (h *Handler) OnDelete(obj interface{}) {
	isync, ok := obj.(*coralv1beta1.ImageSync)
	if !ok {
		h.logger.Error(fmt.Errorf("unexpected type"), "type", reflect.TypeOf(obj))
		return
	}

	h.delete(isync)
}

func (h *Handler) add(obj *coralv1beta1.ImageSync) {
	h.logger.V(2).Info("adding imagesync", "name", obj.Name, "namespace", obj.Namespace)
	h.events <- event.Event{
		Op:     event.Create,
		Object: obj,
	}
}

func (h *Handler) delete(obj *coralv1beta1.ImageSync) {
	h.logger.V(2).Info("deleting imagesync", "name", obj.Name, "namespace", obj.Namespace)
	h.events <- event.Event{
		Op:     event.Delete,
		Object: obj,
	}
}
