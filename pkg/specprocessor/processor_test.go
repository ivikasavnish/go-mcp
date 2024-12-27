// pkg/specprocessor/processor_test.go
package specprocessor

import (
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessor_ProcessOpenAPISpec(t *testing.T) {
	// Create a temporary OpenAPI spec file
	openAPISpec := `{
        "openapi": "3.0.0",
        "info": {
            "title": "Test API",
            "version": "1.0.0"
        },
        "paths": {
            "/test": {
                "get": {
                    "summary": "Test endpoint"
                }
            }
        }
    }`

	tmpDir, err := ioutil.TempDir("", "openapi-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	specPath := filepath.Join(tmpDir, "openapi.json")
	err = ioutil.WriteFile(specPath, []byte(openAPISpec), 0644)
	require.NoError(t, err)

	// Create a test server
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/context/create", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	// Create and use the processor
	processor := NewProcessor(server.URL)
	err = processor.ProcessFile(specPath)
	require.NoError(t, err)

	// Verify the created context
	assert.Equal(t, "openapi-openapi", receivedPayload["id"])
	metadata := receivedPayload["metadata"].(map[string]interface{})
	assert.Equal(t, "openapi", metadata["type"])
	assert.Equal(t, specPath, metadata["source"])
}

func TestProcessor_ProcessPostmanCollection(t *testing.T) {
	// Create a temporary Postman collection file
	postmanCollection := `{
        "info": {
            "name": "Test Collection",
            "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"
        },
        "item": [
            {
                "name": "Test Request",
                "request": {
                    "method": "GET",
                    "url": "http://example.com"
                }
            }
        ]
    }`

	tmpDir, err := ioutil.TempDir("", "postman-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	collectionPath := filepath.Join(tmpDir, "collection.json")
	err = ioutil.WriteFile(collectionPath, []byte(postmanCollection), 0644)
	require.NoError(t, err)

	// Create a test server
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/context/create", r.URL.Path)
		assert.Equal(t, "POST", r.Method)

		err := json.NewDecoder(r.Body).Decode(&receivedPayload)
		require.NoError(t, err)

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	// Create and use the processor
	processor := NewProcessor(server.URL)
	err = processor.ProcessFile(collectionPath)
	require.NoError(t, err)

	// Verify the created context
	assert.Equal(t, "postman-collection", receivedPayload["id"])
	metadata := receivedPayload["metadata"].(map[string]interface{})
	assert.Equal(t, "postman", metadata["type"])
	assert.Equal(t, collectionPath, metadata["source"])
}

func TestProcessor_ProcessDirectory(t *testing.T) {
	// Create a temporary directory with test files
	tmpDir, err := ioutil.TempDir("", "specs-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test files
	files := map[string]string{
		"openapi.json":    `{"openapi": "3.0.0", "info": {"title": "Test API"}}`,
		"collection.json": `{"info": {"name": "Test Collection"}, "item": []}`,
		"openapi.yaml": `openapi: 3.0.0
info:
  title: Test API
`,
	}

	for name, content := range files {
		err = ioutil.WriteFile(filepath.Join(tmpDir, name), []byte(content), 0644)
		require.NoError(t, err)
	}

	// Create a test server
	processedFiles := make(map[string]bool)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		require.NoError(t, err)

		metadata := payload["metadata"].(map[string]interface{})
		source := metadata["source"].(string)
		processedFiles[filepath.Base(source)] = true

		w.WriteHeader(http.StatusCreated)
	}))
	defer server.Close()

	// Create and use the processor
	processor := NewProcessor(server.URL)
	err = processor.ProcessDirectory(tmpDir)
	require.NoError(t, err)

	// Verify all files were processed
	for name := range files {
		assert.True(t, processedFiles[name], "File %s was not processed", name)
	}
}
