package queue

import (
	"strconv"
	"sync"
	"testing"
)

// TestStore_SetGet_Basics проверяет базовую функциональность хранилища состояний
func TestStore_SetGet_Basics(t *testing.T) {
	t.Parallel()

	s := NewStore()
	_, ok := s.GetState("missing")
	if ok {
		t.Fatalf("expected no value")
	}

	s.SetState("a", StateQueued)
	if got, ok := s.GetState("a"); !ok || got != StateQueued {
		t.Fatalf("want %q, ok=true; got %q, ok=%v", StateQueued, got, ok)
	}

	s.SetState("a", StateRunning)
	if got, _ := s.GetState("a"); got != StateRunning {
		t.Fatalf("want %q, got %q", StateRunning, got)
	}
}

// TestStore_Has_Delete проверяет вспомогательные методы Has и Delete.
func TestStore_Has_Delete(t *testing.T) {
	t.Parallel()

	s := NewStore()
	if s.Has("x") {
		t.Fatalf("unexpected Has before set")
	}
	s.SetState("x", StateQueued)
	if !s.Has("x") {
		t.Fatalf("expected Has after set")
	}
	s.Delete("x")
	if s.Has("x") {
		t.Fatalf("expected not Has after delete")
	}
}

// TestStore_ConcurrentAccess_RaceSafe проверяет потокобезопасность.
func TestStore_ConcurrentAccess_RaceSafe(t *testing.T) {
	s := NewStore()

	const goroutines = 32
	const items = 2000

	var wg sync.WaitGroup
	wg.Add(goroutines)

	for g := 0; g < goroutines; g++ {
		go func(id string) {
			defer wg.Done()
			for i := 0; i < items; i++ {
				s.SetState(id, StateQueued)
				if _, ok := s.GetState(id); !ok {
					t.Errorf("GetState(%s) returned ok=false unexpectedly", id)
					return
				}
				s.SetState(id, StateRunning)
				s.SetState(id, StateDone)
			}
		}(strconv.Itoa(g))
	}

	wg.Wait()
}
