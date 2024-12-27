// pkg/mcp/curl_handler.go
package mcp

import (
	"encoding/json"
	"net/http"

	"github.com/ivikasavnish/go-mcp/pkg/curlprocessor"
)

// CurlRequest represents a request to process curl commands
type CurlRequest struct {
	Name     string `json:"name"`
	Commands string `json:"commands"`
}

// AddCurlHandler adds curl processing capabilities to the MCP server
func (s *Server) AddCurlHandler() {
	s.router.HandleFunc("/curl/process", s.handleProcessCurl).Methods("POST")
}

func (s *Server) handleProcessCurl(w http.ResponseWriter, r *http.Request) {
	var req CurlRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	processor := curlprocessor.NewProcessor(s.GetBaseURL())
	if err := processor.ProcessCurlContent(req.Commands, req.Name); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]string{
		"status": "processed",
		"name":   req.Name,
	})
}

// GetBaseURL returns the base URL of the MCP server
func (s *Server) GetBaseURL() string {
	// In a real implementation, this would be configurable
	return "http://localhost:8080"
}
