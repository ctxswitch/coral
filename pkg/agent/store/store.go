package store

import (
	coralctxshv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
	"sync"
	"sync/atomic"
)

type Errors string

func (e Errors) Error() string {
	return string(e)
}

const (
	ErrNotFound Errors = "not found"
)

type Object interface {
	coralctxshv1beta1.ImageSync | ~string
}

type Store[T Object] struct {
	objects map[string]uint32

	sync.Mutex
}

func New[T Object]() *Store[T] {
	return &Store[T]{
		objects: make(map[string]uint32),
	}
}

func (s *Store[T]) References(key string) int {
	s.Lock()
	defer s.Unlock()

	if i, ok := s.objects[key]; ok {
		return int(atomic.LoadUint32(&i))
	}

	return 0
}

func (s *Store[T]) IsReferenced(key string) bool {
	s.Lock()
	defer s.Unlock()

	if o, ok := s.objects[key]; ok && o > 0 {
		return true
	}

	return false
}

func (s *Store[T]) Add(key string) {
	s.Lock()
	defer s.Unlock()

	s.objects[key] += 1
}

func (s *Store[T]) Delete(key string) {
	s.Lock()
	defer s.Unlock()

	if v, ok := s.objects[key]; ok && v > 0 {
		s.objects[key] -= 1
	}

	// TODO: Do I want to error here or potentially delete the element if exists?
}
