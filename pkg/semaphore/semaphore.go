package semaphore

import "sync"

type LockerWithState interface {
	Lock()
	Unlock()
	IsLocked() bool
}

type Semaphore struct {
	mux      *sync.RWMutex
	isLocked bool
}

func NewSemaphore() *Semaphore {
	return &Semaphore{
		mux:      &sync.RWMutex{},
		isLocked: false,
	}
}

func (m *Semaphore) Lock() {
	m.mux.RLock()
	m.isLocked = true
}

func (m *Semaphore) Unlock() {
	m.mux.RUnlock()
	m.isLocked = false
}

func (m *Semaphore) IsLocked() bool {
	m.mux.RLock()
	defer m.mux.RUnlock()
	return m.isLocked
}
