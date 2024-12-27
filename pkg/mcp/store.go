package mcp

import (
	"sync"
)

// Store interface defines the context storage operations
type Store interface {
	Create(*Context) error
	Get(string) (*Context, error)
	Update(*Context) error
	Delete(string) error
	List() []*Context
}

// MemoryStore implements Store interface using in-memory storage
type MemoryStore struct {
	contexts map[string]*Context
	mu       sync.RWMutex
}

// NewMemoryStore creates a new in-memory context store
func NewMemoryStore() Store {
	return &MemoryStore{
		contexts: make(map[string]*Context),
	}
}

func (s *MemoryStore) Create(ctx *Context) error {
	if err := ctx.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.contexts[ctx.ID]; exists {
		return ErrContextExists
	}

	s.contexts[ctx.ID] = ctx.Clone()
	return nil
}

func (s *MemoryStore) Get(id string) (*Context, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ctx, exists := s.contexts[id]
	if !exists {
		return nil, ErrContextNotFound
	}

	return ctx.Clone(), nil
}

func (s *MemoryStore) Update(ctx *Context) error {
	if err := ctx.Validate(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.contexts[ctx.ID]; !exists {
		return ErrContextNotFound
	}

	s.contexts[ctx.ID] = ctx.Clone()
	return nil
}

func (s *MemoryStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.contexts[id]; !exists {
		return ErrContextNotFound
	}

	delete(s.contexts, id)
	return nil
}

func (s *MemoryStore) List() []*Context {
	s.mu.RLock()
	defer s.mu.RUnlock()

	contexts := make([]*Context, 0, len(s.contexts))
	for _, ctx := range s.contexts {
		contexts = append(contexts, ctx.Clone())
	}
	return contexts
}
