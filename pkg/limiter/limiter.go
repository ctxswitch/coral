package limiter

type Limiter struct {
	events chan int
}

func New(max int) *Limiter {
	return &Limiter{
		events: make(chan int, max),
	}
}

func (l *Limiter) Acquire() {
	l.events <- 1
}

func (l *Limiter) Release() {
	<-l.events
}
