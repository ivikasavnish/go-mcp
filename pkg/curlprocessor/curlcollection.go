// pkg/curlprocessor/parser.go
package curlprocessor

import (
	"fmt"
	"net/url"
	"strings"
)

// CurlCommand represents a parsed curl command
type CurlCommand struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	QueryParams url.Values        `json:"query_params"`
	Auth        *Authentication   `json:"auth,omitempty"`
}

// Authentication represents authentication details
type Authentication struct {
	Type     string `json:"type"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
	Token    string `json:"token,omitempty"`
}

// CurlCollection represents a collection of curl commands
type CurlCollection struct {
	Name     string        `json:"name"`
	Commands []CurlCommand `json:"commands"`
}

// ParseCurlCommand parses a curl command string into a structured format
func ParseCurlCommand(cmd string) (*CurlCommand, error) {
	curl := &CurlCommand{
		Method:      "GET",
		Headers:     make(map[string]string),
		QueryParams: make(url.Values),
	}

	// Remove 'curl' prefix if present
	cmd = strings.TrimPrefix(cmd, "curl")
	cmd = strings.TrimSpace(cmd)

	// Split the command into parts while preserving quoted strings
	parts := splitCommand(cmd)

	for i := 0; i < len(parts); i++ {
		part := parts[i]
		switch {
		case part == "-X" || part == "--request":
			if i+1 < len(parts) {
				curl.Method = parts[i+1]
				i++
			}
		case part == "-H" || part == "--header":
			if i+1 < len(parts) {
				header := parts[i+1]
				if key, value, ok := parseHeader(header); ok {
					curl.Headers[key] = value
				}
				i++
			}
		case part == "-d" || part == "--data" || part == "--data-raw":
			if i+1 < len(parts) {
				curl.Body = parts[i+1]
				if curl.Method == "GET" {
					curl.Method = "POST"
				}
				i++
			}
		case part == "-u" || part == "--user":
			if i+1 < len(parts) {
				auth := parts[i+1]
				if username, password, ok := parseAuth(auth); ok {
					curl.Auth = &Authentication{
						Type:     "basic",
						Username: username,
						Password: password,
					}
				}
				i++
			}
		case strings.HasPrefix(part, "http://") || strings.HasPrefix(part, "https://"):
			curl.URL = part
			if u, err := url.Parse(part); err == nil {
				curl.QueryParams = u.Query()
			}
		}
	}

	if curl.URL == "" {
		return nil, fmt.Errorf("no URL found in curl command")
	}

	return curl, nil
}

// Helper functions

func splitCommand(cmd string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, ch := range cmd {
		switch {
		case ch == '"' || ch == '\'':
			if !inQuote {
				inQuote = true
				quoteChar = ch
			} else if ch == quoteChar {
				inQuote = false
				quoteChar = rune(0)
			} else {
				current.WriteRune(ch)
			}
		case ch == ' ' && !inQuote:
			if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}

	if current.Len() > 0 {
		parts = append(parts, current.String())
	}

	return parts
}

func parseHeader(header string) (string, string, bool) {
	parts := strings.SplitN(header, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}

func parseAuth(auth string) (string, string, bool) {
	parts := strings.SplitN(auth, ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}

// ParseCurlCollection parses multiple curl commands from a string
func ParseCurlCollection(content string, name string) (*CurlCollection, error) {
	collection := &CurlCollection{
		Name:     name,
		Commands: make([]CurlCommand, 0),
	}

	// Split content into individual commands
	commands := splitCommands(content)

	for _, cmd := range commands {
		if curlCmd, err := ParseCurlCommand(cmd); err == nil {
			collection.Commands = append(collection.Commands, *curlCmd)
		}
	}

	if len(collection.Commands) == 0 {
		return nil, fmt.Errorf("no valid curl commands found")
	}

	return collection, nil
}

func splitCommands(content string) []string {
	// Split on newlines and remove empty lines
	lines := strings.Split(content, "\n")
	commands := make([]string, 0)
	var currentCmd strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// If line ends with backslash, it's a continuation
		if strings.HasSuffix(line, "\\") {
			currentCmd.WriteString(strings.TrimSuffix(line, "\\"))
			currentCmd.WriteString(" ")
		} else {
			currentCmd.WriteString(line)
			if currentCmd.Len() > 0 {
				commands = append(commands, currentCmd.String())
			}
			currentCmd.Reset()
		}
	}

	return commands
}
