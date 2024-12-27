// pkg/specprocessor/processor.go
package specprocessor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// MCPClient handles communication with the MCP server
type MCPClient struct {
	baseURL string
	client  *http.Client
}

// NewMCPClient creates a new MCP client
func NewMCPClient(baseURL string) *MCPClient {
	return &MCPClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

// CreateContext sends a request to create a new context in MCP
func (c *MCPClient) CreateContext(id string, metadata map[string]interface{}) error {
	payload := map[string]interface{}{
		"id":       id,
		"metadata": metadata,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed to marshal context data: %w", err)
	}

	resp, err := c.client.Post(
		fmt.Sprintf("%s/context/create", c.baseURL),
		"application/json",
		bytes.NewBuffer(jsonData),
	)
	if err != nil {
		return fmt.Errorf("failed to create context: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("failed to create context, status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Processor handles processing of API specifications
type Processor struct {
	mcpClient *MCPClient
	logger    *log.Logger
}

// ProcessorOption defines options for creating a new Processor
type ProcessorOption func(*Processor)

// WithLogger sets a custom logger for the processor
func WithLogger(logger *log.Logger) ProcessorOption {
	return func(p *Processor) {
		p.logger = logger
	}
}

// NewProcessor creates a new specification processor
func NewProcessor(mcpBaseURL string, opts ...ProcessorOption) *Processor {
	p := &Processor{
		mcpClient: NewMCPClient(mcpBaseURL),
		logger:    log.New(ioutil.Discard, "", 0),
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

// ProcessFile processes a single specification file
func (p *Processor) ProcessFile(filePath string) error {
	ext := strings.ToLower(filepath.Ext(filePath))

	p.logger.Printf("Processing file: %s", filePath)

	var err error
	switch ext {
	case ".json":
		err = p.ProcessPostmanCollection(filePath)
		if err != nil {
			// If it's not a Postman collection, try processing as OpenAPI
			err = p.ProcessOpenAPISpec(filePath)
		}
	case ".yaml", ".yml":
		err = p.ProcessOpenAPISpec(filePath)
	default:
		return fmt.Errorf("unsupported file type: %s", ext)
	}

	if err != nil {
		return fmt.Errorf("failed to process %s: %w", filePath, err)
	}

	return nil
}

// ProcessDirectory processes all API specifications in a directory
func (p *Processor) ProcessDirectory(dirPath string) error {
	p.logger.Printf("Processing directory: %s", dirPath)

	files, err := ioutil.ReadDir(dirPath)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filePath := filepath.Join(dirPath, file.Name())
		if err := p.ProcessFile(filePath); err != nil {
			p.logger.Printf("Error processing file %s: %v", filePath, err)
			continue
		}
	}

	return nil
}

// ProcessOpenAPISpec processes an OpenAPI specification file
func (p *Processor) ProcessOpenAPISpec(filePath string) error {
	p.logger.Printf("Processing OpenAPI spec: %s", filePath)

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read OpenAPI spec file: %w", err)
	}

	var spec map[string]interface{}
	ext := strings.ToLower(filepath.Ext(filePath))

	if ext == ".yaml" || ext == ".yml" {
		err = yaml.Unmarshal(data, &spec)
	} else {
		err = json.Unmarshal(data, &spec)
	}

	if err != nil {
		return fmt.Errorf("failed to parse OpenAPI spec: %w", err)
	}

	// Validate that it's actually an OpenAPI spec
	if _, ok := spec["openapi"]; !ok {
		return fmt.Errorf("not a valid OpenAPI specification")
	}

	metadata := map[string]interface{}{
		"type":   "openapi",
		"spec":   spec,
		"source": filePath,
	}

	contextID := fmt.Sprintf("openapi-%s", strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)))
	return p.mcpClient.CreateContext(contextID, metadata)
}

// ProcessPostmanCollection processes a Postman collection file
func (p *Processor) ProcessPostmanCollection(filePath string) error {
	p.logger.Printf("Processing Postman collection: %s", filePath)

	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read Postman collection file: %w", err)
	}

	var collection map[string]interface{}
	if err := json.Unmarshal(data, &collection); err != nil {
		return fmt.Errorf("failed to parse Postman collection: %w", err)
	}

	// Validate that it's actually a Postman collection
	if _, ok := collection["info"]; !ok {
		return fmt.Errorf("not a valid Postman collection")
	}

	metadata := map[string]interface{}{
		"type":       "postman",
		"collection": collection,
		"source":     filePath,
	}

	contextID := fmt.Sprintf("postman-%s", strings.TrimSuffix(filepath.Base(filePath), filepath.Ext(filePath)))
	return p.mcpClient.CreateContext(contextID, metadata)
}
