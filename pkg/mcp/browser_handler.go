package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/mux"
	"github.com/ivikasavnish/go-mcp/pkg/browser"
)

// BrowserManager manages browser instances
type BrowserManager struct {
	browsers map[string]*browser.Browser
	mu       sync.RWMutex
}

func NewBrowserManager() *BrowserManager {
	return &BrowserManager{
		browsers: make(map[string]*browser.Browser),
	}
}

// Request/Response types
type CreateBrowserRequest struct {
	ID     string                `json:"id"`
	Config browser.BrowserConfig `json:"config"`
}

type NavigateRequest struct {
	URL string `json:"url"`
}

type ScrapingRequest struct {
	Selectors map[string]string `json:"selectors"`
}

type ScreenshotRequest struct {
	FullPage bool   `json:"full_page"`
	Format   string `json:"format"`
}

type AutomationRequest struct {
	Sequence browser.AutomationSequence `json:"sequence"`
}

// AddBrowserHandlers adds browser automation endpoints to the MCP server
func (s *Server) AddBrowserHandlers() {
	manager := NewBrowserManager()

	// Browser instance management
	s.router.HandleFunc("/browser/create", handleCreateBrowser(manager)).Methods("POST")
	s.router.HandleFunc("/browser/{id}", handleCloseBrowser(manager)).Methods("DELETE")

	// Navigation and automation
	s.router.HandleFunc("/browser/{id}/navigate", handleNavigate(manager)).Methods("POST")
	s.router.HandleFunc("/browser/{id}/automate", handleAutomate(manager)).Methods("POST")
	//s.router.HandleFunc("/browser/{id}/scrape", handleScrape(manager)).Methods("POST")
	//s.router.HandleFunc("/browser/{id}/screenshot", handleScreenshot(manager)).Methods("POST")
}

func handleCreateBrowser(bm *BrowserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateBrowserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		bm.mu.Lock()
		if _, exists := bm.browsers[req.ID]; exists {
			bm.mu.Unlock()
			writeError(w, http.StatusConflict, fmt.Errorf("browser with ID %s already exists", req.ID))
			return
		}

		b := browser.NewBrowser(&req.Config)
		if err := b.Start(); err != nil {
			bm.mu.Unlock()
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		bm.browsers[req.ID] = b
		bm.mu.Unlock()

		writeJSON(w, http.StatusCreated, map[string]string{
			"id":     req.ID,
			"status": "created",
		})
	}
}

func handleCloseBrowser(bm *BrowserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		bm.mu.Lock()
		b, exists := bm.browsers[id]
		if !exists {
			bm.mu.Unlock()
			writeError(w, http.StatusNotFound, fmt.Errorf("browser not found"))
			return
		}

		if err := b.Stop(); err != nil {
			bm.mu.Unlock()
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		delete(bm.browsers, id)
		bm.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]string{
			"id":     id,
			"status": "closed",
		})
	}
}

func handleNavigate(bm *BrowserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		var req NavigateRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		bm.mu.RLock()
		b, exists := bm.browsers[id]
		bm.mu.RUnlock()

		if !exists {
			writeError(w, http.StatusNotFound, fmt.Errorf("browser not found"))
			return
		}

		result, err := b.Navigate(req.URL)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func handleAutomate(bm *BrowserManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := mux.Vars(r)["id"]

		var req AutomationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		bm.mu.RLock()
		b, exists := bm.browsers[id]
		bm.mu.RUnlock()

		if !exists {
			writeError(w, http.StatusNotFound, fmt.Errorf("browser not found"))
			return
		}

		if err := b.ExecuteSequence(&req.Sequence); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "completed",
		})

	}
}
