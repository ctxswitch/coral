package queue

type Queue struct {
	events chan int
}

func New(max int) *Queue {
	return &Queue{
		events: make(chan int, max),
	}
}

func (q *Queue) Acquire() {
	q.events <- 1
}

func (q *Queue) Release() {
	<-q.events
}
