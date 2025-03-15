package event

import "sigs.k8s.io/controller-runtime/pkg/client"

type Operation int

const (
	Create Operation = iota
	Delete
	Update
)

type Event struct {
	Op     Operation
	Object client.Object
}

func NewEvent(obj client.Object, op Operation) *Event {
	return &Event{
		Op:     op,
		Object: obj,
	}
}

func (e *Event) GetOperationString() string {
	return []string{
		"create",
		"delete",
		"update",
	}[e.Op]
}
