package mcp

import (
	"encoding/json"
	"fmt"
	"github.com/gorilla/mux"
	"net/http"
	"sync"
)

// SSHManager manages SSH connections
type SSHManager struct {
	clients map[string]*SSHClient
	mu      sync.RWMutex
}

// SSHConnectionRequest represents an SSH connection request
type SSHConnectionRequest struct {
	ID     string    `json:"id"`
	Config SSHConfig `json:"config"`
}

// SSHCommandRequest represents an SSH command execution request
type SSHCommandRequest struct {
	Command string `json:"command"`
}

// SSHFileTransferRequest represents a file transfer request
type SSHFileTransferRequest struct {
	LocalPath  string `json:"local_path"`
	RemotePath string `json:"remote_path"`
}

// NewSSHManager creates a new SSH manager
func NewSSHManager() *SSHManager {
	return &SSHManager{
		clients: make(map[string]*SSHClient),
	}
}

// AddSSHHandler adds SSH handling capabilities to the MCP server
func (s *Server) AddSSHHandler() {
	manager := NewSSHManager()

	// Connection management
	s.router.HandleFunc("/ssh/connect", handleSSHConnect(manager)).Methods("POST")
	s.router.HandleFunc("/ssh/{id}", handleSSHDisconnect(manager)).Methods("DELETE")

	// Command execution
	s.router.HandleFunc("/ssh/{id}/exec", handleSSHExec(manager)).Methods("POST")

	// File transfer
	s.router.HandleFunc("/ssh/{id}/upload", handleSSHUpload(manager)).Methods("POST")
	s.router.HandleFunc("/ssh/{id}/download", handleSSHDownload(manager)).Methods("POST")
}

func handleSSHConnect(manager *SSHManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req SSHConnectionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		manager.mu.Lock()
		if _, exists := manager.clients[req.ID]; exists {
			manager.mu.Unlock()
			writeError(w, http.StatusConflict, fmt.Errorf("connection with ID %s already exists", req.ID))
			return
		}

		client, err := NewSSHClient(req.Config)
		if err != nil {
			manager.mu.Unlock()
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		if err := client.Connect(); err != nil {
			manager.mu.Unlock()
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		manager.clients[req.ID] = client
		manager.mu.Unlock()

		writeJSON(w, http.StatusCreated, map[string]string{
			"id":     req.ID,
			"status": "connected",
		})
	}
}

func handleSSHDisconnect(manager *SSHManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		manager.mu.Lock()
		client, exists := manager.clients[id]
		if !exists {
			manager.mu.Unlock()
			writeError(w, http.StatusNotFound, fmt.Errorf("connection not found"))
			return
		}

		if err := client.Close(); err != nil {
			manager.mu.Unlock()
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		delete(manager.clients, id)
		manager.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]string{
			"id":     id,
			"status": "disconnected",
		})
	}
}

func handleSSHExec(manager *SSHManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		var req SSHCommandRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		manager.mu.RLock()
		client, exists := manager.clients[id]
		manager.mu.RUnlock()

		if !exists {
			writeError(w, http.StatusNotFound, fmt.Errorf("connection not found"))
			return
		}

		result, err := client.ExecuteCommand(req.Command)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func handleSSHUpload(manager *SSHManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		var req SSHFileTransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		manager.mu.RLock()
		client, exists := manager.clients[id]
		manager.mu.RUnlock()

		if !exists {
			writeError(w, http.StatusNotFound, fmt.Errorf("connection not found"))
			return
		}

		if err := client.UploadFile(req.LocalPath, req.RemotePath); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "uploaded",
			"path":   req.RemotePath,
		})
	}
}

func handleSSHDownload(manager *SSHManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		var req SSHFileTransferRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		manager.mu.RLock()
		client, exists := manager.clients[id]
		manager.mu.RUnlock()

		if !exists {
			writeError(w, http.StatusNotFound, fmt.Errorf("connection not found"))
			return
		}

		if err := client.DownloadFile(req.RemotePath, req.LocalPath); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "downloaded",
			"path":   req.LocalPath,
		})
	}
}
