package imagesync

type LabelOperation int

const (
	Add LabelOperation = iota
	Remove
)

type LabelEvent struct {
	Op    LabelOperation
	Key   string
	Value string
}
