// pkg/curlprocessor/processor.go
package curlprocessor

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/ivikasavnish/go-mcp/pkg/specprocessor"
)

// Processor processes curl collections and integrates with MCP
type Processor struct {
	mcpClient *specprocessor.MCPClient
}

// NewProcessor creates a new curl processor
func NewProcessor(mcpBaseURL string) *Processor {
	return &Processor{
		mcpClient: specprocessor.NewMCPClient(mcpBaseURL),
	}
}

// ProcessCurlFile processes a file containing curl commands
func (p *Processor) ProcessCurlFile(filePath string) error {
	content, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read curl file: %w", err)
	}

	name := strings.TrimSuffix(filePath, ".txt")
	collection, err := ParseCurlCollection(string(content), name)
	if err != nil {
		return fmt.Errorf("failed to parse curl collection: %w", err)
	}

	return p.createMCPContext(collection, filePath)
}

// ProcessCurlContent processes curl commands from a string
func (p *Processor) ProcessCurlContent(content, name string) error {
	collection, err := ParseCurlCollection(content, name)
	if err != nil {
		return fmt.Errorf("failed to parse curl collection: %w", err)
	}

	return p.createMCPContext(collection, "inline")
}

func (p *Processor) createMCPContext(collection *CurlCollection, source string) error {
	metadata := map[string]interface{}{
		"type":       "curl",
		"collection": collection,
		"source":     source,
		"timestamp":  time.Now(),
	}

	contextID := fmt.Sprintf("curl-%s", strings.ReplaceAll(collection.Name, " ", "-"))
	return p.mcpClient.CreateContext(contextID, metadata)
}
