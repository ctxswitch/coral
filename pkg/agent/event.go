package agent

import "sigs.k8s.io/controller-runtime/pkg/client"

type Operation int

const (
	Create Operation = iota
	Delete
	Update
)

type Event struct {
	op  Operation
	obj client.Object
}

func NewEvent(obj client.Object, op Operation) *Event {
	return &Event{
		op:  op,
		obj: obj,
	}
}
