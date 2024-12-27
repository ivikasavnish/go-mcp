// pkg/mcp/server.go
package mcp

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

// Server represents the MCP server
type Server struct {
	store  Store
	router *mux.Router
}

// NewServer creates a new MCP server instance
func NewServer(store Store) *Server {
	if store == nil {
		store = NewMemoryStore()
	}

	s := &Server{
		store:  store,
		router: mux.NewRouter(),
	}

	s.setupRoutes()
	return s
}

func (s *Server) setupRoutes() {
	s.router.HandleFunc("/context/create", s.handleCreateContext).Methods("POST")
	s.router.HandleFunc("/context/get", s.handleGetContext).Methods("GET")
	s.router.HandleFunc("/context/update", s.handleUpdateContext).Methods("PUT")
	s.router.HandleFunc("/context/delete", s.handleDeleteContext).Methods("DELETE")
	s.router.HandleFunc("/context/list", s.handleListContexts).Methods("GET")
}

// ServeHTTP implements the http.Handler interface
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.router.ServeHTTP(w, r)
}

// Start starts the server on the specified address
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s)
}

// Request/Response types
type CreateContextRequest struct {
	ID       string                 `json:"id"`
	Metadata map[string]interface{} `json:"metadata"`
}

type UpdateContextRequest struct {
	Metadata map[string]interface{} `json:"metadata"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, ErrorResponse{Error: err.Error()})
}

func (s *Server) handleCreateContext(w http.ResponseWriter, r *http.Request) {
	var req CreateContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ctx := &Context{
		ID:        req.ID,
		Metadata:  req.Metadata,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.store.Create(ctx); err != nil {
		status := http.StatusInternalServerError
		if err == ErrContextExists {
			status = http.StatusConflict
		} else if err == ErrInvalidID || err == ErrInvalidMetadata {
			status = http.StatusBadRequest
		}
		writeError(w, status, err)
		return
	}

	writeJSON(w, http.StatusCreated, ctx)
}

func (s *Server) handleGetContext(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, ErrInvalidID)
		return
	}

	ctx, err := s.store.Get(id)
	if err != nil {
		status := http.StatusInternalServerError
		if err == ErrContextNotFound {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	writeJSON(w, http.StatusOK, ctx)
}

func (s *Server) handleUpdateContext(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, ErrInvalidID)
		return
	}

	var req UpdateContextRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ctx, err := s.store.Get(id)
	if err != nil {
		status := http.StatusInternalServerError
		if err == ErrContextNotFound {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	ctx.Metadata = req.Metadata
	ctx.UpdatedAt = time.Now()

	if err := s.store.Update(ctx); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, ctx)
}

func (s *Server) handleDeleteContext(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, ErrInvalidID)
		return
	}

	if err := s.store.Delete(id); err != nil {
		status := http.StatusInternalServerError
		if err == ErrContextNotFound {
			status = http.StatusNotFound
		}
		writeError(w, status, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListContexts(w http.ResponseWriter, r *http.Request) {
	contexts := s.store.List()
	writeJSON(w, http.StatusOK, contexts)
}
