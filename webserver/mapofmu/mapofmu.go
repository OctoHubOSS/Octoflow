//  Copyright (C) 2026 NodeByte LTD

package mapofmu

import (
	"fmt"
	"sync"
)

type M[K comparable] struct {
	ml sync.Mutex       // lock for entry map
	ma map[K]*mentry[K] // entry map
}

type mentry[K comparable] struct {
	m   *M[K]
	el  sync.Mutex
	cnt int
	key K
}

type Unlocker interface {
	Unlock()
}

func New[K comparable]() *M[K] {
	return &M[K]{ma: make(map[K]*mentry[K])}
}

func (m *M[K]) Lock(key K) Unlocker {
	m.ml.Lock()
	e, ok := m.ma[key]
	if !ok {
		e = &mentry[K]{m: m, key: key}
		m.ma[key] = e
	}
	e.cnt++
	m.ml.Unlock()
	e.el.Lock()

	return e
}

func (me *mentry[K]) Unlock() {

	m := me.m

	m.ml.Lock()
	e, ok := m.ma[me.key]
	if !ok {
		m.ml.Unlock()
		panic(fmt.Errorf("Unlock requested for key=%v but no entry found", me.key))
	}
	e.cnt--
	if e.cnt < 1 {
		delete(m.ma, me.key)
	}
	m.ml.Unlock()

	e.el.Unlock()

}
