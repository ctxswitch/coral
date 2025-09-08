// Copyright 2025 Coral Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package store

import (
	"sync"
	"sync/atomic"

	coralctxshv1beta1 "ctx.sh/coral/pkg/apis/coral.ctx.sh/v1beta1"
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
