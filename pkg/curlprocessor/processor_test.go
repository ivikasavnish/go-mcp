// pkg/curlprocessor/processor_test.go
package curlprocessor

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCurlCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected *CurlCommand
		wantErr  bool
	}{
		{
			name:    "Simple GET request",
			command: `curl https://api.example.com/users`,
			expected: &CurlCommand{
				Method:  "GET",
				URL:     "https://api.example.com/users",
				Headers: make(map[string]string),
			},
			wantErr: false,
		},
		{
			name:    "POST request with headers and body",
			command: `curl -X POST -H "Content-Type: application/json" -d '{"name":"test"}' https://api.example.com/users`,
			expected: &CurlCommand{
				Method: "POST",
				URL:    "https://api.example.com/users",
				Headers: map[string]string{
					"Content-Type": "application/json",
				},
				Body: `{"name":"test"}`,
			},
			wantErr: false,
		},
		{
			name:    "Request with basic auth",
			command: `curl -u username:password https://api.example.com/secure`,
			expected: &CurlCommand{
				Method:  "GET",
				URL:     "https://api.example.com/secure",
				Headers: make(map[string]string),
				Auth: &Authentication{
					Type:     "basic",
					Username: "username",
					Password: "password",
				},
			},
			wantErr: false,
		},
		{
			name:    "Request with query parameters",
			command: `curl "https://api.example.com/search?q=test&page=1"`,
			expected: &CurlCommand{
				Method:  "GET",
				URL:     "https://api.example.com/search?q=test&page=1",
				Headers: make(map[string]string),
				QueryParams: map[string][]string{
					"q":    {"test"},
					"page": {"1"},
				},
			},
			wantErr: false,
		},
		{
			name:    "Request with multiple headers",
			command: `curl -H "Content-Type: application/json" -H "Authorization: Bearer token123" https://api.example.com/data`,
			expected: &CurlCommand{
				Method: "GET",
				URL:    "https://api.example.com/data",
				Headers: map[string]string{
					"Content-Type":  "application/json",
					"Authorization": "Bearer token123",
				},
			},
			wantErr: false,
		},
		{
			name:    "Invalid command - no URL",
			command: `curl -X POST -H "Content-Type: application/json"`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := ParseCurlCommand(tt.command)
			if tt.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.expected.Method, cmd.Method)
			assert.Equal(t, tt.expected.URL, cmd.URL)
			assert.Equal(t, tt.expected.Headers, cmd.Headers)
			assert.Equal(t, tt.expected.Body, cmd.Body)
			if tt.expected.Auth != nil {
				require.NotNil(t, cmd.Auth)
				assert.Equal(t, tt.expected.Auth.Type, cmd.Auth.Type)
				assert.Equal(t, tt.expected.Auth.Username, cmd.Auth.Username)
				assert.Equal(t, tt.expected.Auth.Password, cmd.Auth.Password)
			}
			if len(tt.expected.QueryParams) > 0 {
				assert.Equal(t, tt.expected.QueryParams, cmd.QueryParams)
			}
		})
	}
}

func TestParseCurlCollection(t *testing.T) {
	content := `
curl https://api.example.com/users

curl -X POST -H "Content-Type: application/json" \
     -d '{"name":"test"}' \
     https://api.example.com/users

curl -u username:password https://api.example.com/secure
`

	collection, err := ParseCurlCollection(content, "Test Collection")
	require.NoError(t, err)

	assert.Equal(t, "Test Collection", collection.Name)
	assert.Len(t, collection.Commands, 3)

	// Verify first command
	assert.Equal(t, "GET", collection.Commands[0].Method)
	assert.Equal(t, "https://api.example.com/users", collection.Commands[0].URL)

	// Verify second command
	assert.Equal(t, "POST", collection.Commands[1].Method)
	assert.Equal(t, "https://api.example.com/users", collection.Commands[1].URL)
	assert.Equal(t, `{"name":"test"}`, collection.Commands[1].Body)
	assert.Equal(t, "application/json", collection.Commands[1].Headers["Content-Type"])

	// Verify third command
	assert.Equal(t, "GET", collection.Commands[2].Method)
	assert.Equal(t, "https://api.example.com/secure", collection.Commands[2].URL)
	assert.Equal(t, "basic", collection.Commands[2].Auth.Type)
	assert.Equal(t, "username", collection.Commands[2].Auth.Username)
	assert.Equal(t, "password", collection.Commands[2].Auth.Password)
}

func TestProcessor(t *testing.T) {
	// Create a test MCP server
	var receivedContext map[string]interface{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/context/create" {
			err := json.NewDecoder(r.Body).Decode(&receivedContext)
			require.NoError(t, err)
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer testServer.Close()

	// Create temporary curl file
	tmpDir, err := os.MkdirTemp("", "curl-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	curlFile := filepath.Join(tmpDir, "commands.txt")
	content := `
curl https://api.example.com/users
curl -X POST -H "Content-Type: application/json" -d '{"name":"test"}' https://api.example.com/users
`
	err = os.WriteFile(curlFile, []byte(content), 0644)
	require.NoError(t, err)

	// Test file processing
	t.Run("Process curl file", func(t *testing.T) {
		processor := NewProcessor(testServer.URL)
		err := processor.ProcessCurlFile(curlFile)
		require.NoError(t, err)

		require.NotNil(t, receivedContext)
		metadata, ok := receivedContext["metadata"].(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, "curl", metadata["type"])
		assert.Equal(t, curlFile, metadata["source"])

		collection := metadata["collection"].(map[string]interface{})
		commands := collection["commands"].([]interface{})
		assert.Len(t, commands, 2)
	})

	// Test direct content processing
	t.Run("Process curl content", func(t *testing.T) {
		processor := NewProcessor(testServer.URL)
		err := processor.ProcessCurlContent(content, "Test Commands")
		require.NoError(t, err)

		require.NotNil(t, receivedContext)
		metadata, ok := receivedContext["metadata"].(map[string]interface{})
		require.True(t, ok)

		assert.Equal(t, "curl", metadata["type"])
		assert.Equal(t, "inline", metadata["source"])

		collection := metadata["collection"].(map[string]interface{})
		assert.Equal(t, "Test Commands", collection["name"])
	})

	// Test error cases
	t.Run("Process invalid file", func(t *testing.T) {
		processor := NewProcessor(testServer.URL)
		err := processor.ProcessCurlFile("nonexistent.txt")
		require.Error(t, err)
	})

	t.Run("Process empty content", func(t *testing.T) {
		processor := NewProcessor(testServer.URL)
		err := processor.ProcessCurlContent("", "Empty Test")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no valid curl commands found")
	})

	t.Run("Process invalid curl commands", func(t *testing.T) {
		processor := NewProcessor(testServer.URL)
		err := processor.ProcessCurlContent("invalid command\nmore invalid", "Invalid Test")
		require.Error(t, err)
	})
}
