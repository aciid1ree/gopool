package queue

import "sync"

type Store interface {
	SetState(id string, s State)
	GetState(id string) (State, bool)
	Has(id string) bool
	Delete(id string) // optional helper; not required for basic flow
}

// memStore — простая реализация в памяти, использующая словарь, защищенный мьютексом.
type memStore struct {
	mu   sync.Mutex
	data map[string]State
}

func NewStore() Store {
	return &memStore{
		data: make(map[string]State, 128),
	}
}

func (m *memStore) SetState(id string, s State) {
	m.mu.Lock()
	m.data[id] = s
	m.mu.Unlock()
}

func (m *memStore) GetState(id string) (State, bool) {
	m.mu.Lock()
	s, ok := m.data[id]
	m.mu.Unlock()
	return s, ok
}

func (m *memStore) Has(id string) bool {
	m.mu.Lock()
	_, ok := m.data[id]
	m.mu.Unlock()
	return ok
}

func (m *memStore) Delete(id string) {
	m.mu.Lock()
	delete(m.data, id)
	m.mu.Unlock()
}
