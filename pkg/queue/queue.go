package queue

type Queue struct {
	events chan int
}

func New(max uint32) *Queue {
	return &Queue{
		events: make(chan int, max),
	}
}

func (q *Queue) Aquire() {
	q.events <- 1
}

func (q *Queue) Release() {
	<-q.events
}
