package mcp

import (
	"errors"
	"regexp"
	"time"
)

var (
	ErrContextNotFound = errors.New("context not found")
	ErrContextExists   = errors.New("context already exists")
	ErrInvalidID       = errors.New("invalid context ID")
	ErrInvalidMetadata = errors.New("invalid metadata")
)

var validIDPattern = regexp.MustCompile(`^[a-zA-Z0-9-_]+$`)

// Context represents a model context with metadata
type Context struct {
	ID        string                 `json:"id"`
	Metadata  map[string]interface{} `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
}

// Validate checks if the context is valid
func (c *Context) Validate() error {
	if !validIDPattern.MatchString(c.ID) {
		return ErrInvalidID
	}
	if c.Metadata == nil {
		return ErrInvalidMetadata
	}
	return nil
}

// Clone creates a deep copy of the context
func (c *Context) Clone() *Context {
	metadata := make(map[string]interface{})
	for k, v := range c.Metadata {
		metadata[k] = v
	}
	return &Context{
		ID:        c.ID,
		Metadata:  metadata,
		CreatedAt: c.CreatedAt,
		UpdatedAt: c.UpdatedAt,
	}
}
