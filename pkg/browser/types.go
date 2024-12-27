package browser

import (
	"time"
)

// BrowserConfig represents browser configuration options
type BrowserConfig struct {
	Headless  bool              `json:"headless"`
	UserAgent string            `json:"user_agent,omitempty"`
	ViewPort  *ViewPort         `json:"viewport,omitempty"`
	Headers   map[string]string `json:"headers,omitempty"`
	Proxy     string            `json:"proxy,omitempty"`
	Timeout   time.Duration     `json:"timeout,omitempty"`
}

// ViewPort represents browser viewport settings
type ViewPort struct {
	Width  int `json:"width"`
	Height int `json:"height"`
}

// NavigationResult represents the result of a page navigation
type NavigationResult struct {
	Success      bool    `json:"success"`
	URL          string  `json:"url"`
	Title        string  `json:"title"`
	LoadTime     float64 `json:"load_time"`
	StatusCode   int     `json:"status_code"`
	ErrorMessage string  `json:"error_message,omitempty"`
}

// ScrapingResult represents scraped data from a page
type ScrapingResult struct {
	URL       string                 `json:"url"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Screenshot represents a captured screenshot
type Screenshot struct {
	Data      []byte    `json:"-"`
	Format    string    `json:"format"`
	FullPage  bool      `json:"full_page"`
	Timestamp time.Time `json:"timestamp"`
}

// AutomationStep represents a single automation step
type AutomationStep struct {
	Type    string                 `json:"type"`
	Params  map[string]interface{} `json:"params"`
	Timeout time.Duration          `json:"timeout,omitempty"`
	Wait    time.Duration          `json:"wait,omitempty"`
}

// AutomationSequence represents a sequence of automation steps
type AutomationSequence struct {
	Name        string           `json:"name"`
	Description string           `json:"description,omitempty"`
	Steps       []AutomationStep `json:"steps"`
	Config      *BrowserConfig   `json:"config,omitempty"`
}
