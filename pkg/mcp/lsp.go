package mcp

import (
	"encoding/json"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"net/http"
	"strings"
	"sync"
)

// LanguageServer handles LSP functionality
type LanguageServer struct {
	workspaceRoot string
	documents     map[string]*Document
	fileSet       *token.FileSet
	mu            sync.RWMutex
}

// Document represents a source code document
type Document struct {
	URI     string       `json:"uri"`
	Text    string       `json:"text"`
	AST     *ast.File    `json:"ast,omitempty"`
	Symbols []SymbolInfo `json:"symbols"`
	Version int          `json:"version"`
}

// SymbolInfo represents a code symbol (function, type, variable, etc.)
type SymbolInfo struct {
	Name      string   `json:"name"`
	Kind      string   `json:"kind"` // function, type, variable, etc.
	Location  Location `json:"location"`
	Container string   `json:"container,omitempty"`
	Signature string   `json:"signature,omitempty"`
}

// Location represents a position in a document
type Location struct {
	URI   string `json:"uri"`
	Range Range  `json:"range"`
}

// Range represents a text range with start and end positions
type Range struct {
	Start Position `json:"start"`
	End   Position `json:"end"`
}

// Position represents a position in a document
type Position struct {
	Line      int `json:"line"`
	Character int `json:"character"`
}

// LSPRequest represents an incoming LSP request
type LSPRequest struct {
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

// TextDocumentItem represents a text document
type TextDocumentItem struct {
	URI     string `json:"uri"`
	Text    string `json:"text"`
	Version int    `json:"version"`
}

// NewLanguageServer creates a new language server instance
func NewLanguageServer(workspaceRoot string) *LanguageServer {
	return &LanguageServer{
		workspaceRoot: workspaceRoot,
		documents:     make(map[string]*Document),
		fileSet:       token.NewFileSet(),
	}
}

// AddLanguageServerHandler adds LSP capabilities to the MCP server
func (s *Server) AddLanguageServerHandler() {
	ls := NewLanguageServer(s.GetWorkspaceRoot())

	// Document management
	s.router.HandleFunc("/lsp/document/open", handleOpenDocument(ls)).Methods("POST")
	s.router.HandleFunc("/lsp/document/close", handleCloseDocument(ls)).Methods("POST")
	s.router.HandleFunc("/lsp/document/change", handleChangeDocument(ls)).Methods("POST")

	// Code intelligence
	s.router.HandleFunc("/lsp/symbols", handleDocumentSymbols(ls)).Methods("GET")
	s.router.HandleFunc("/lsp/completion", handleCompletion(ls)).Methods("GET")
	s.router.HandleFunc("/lsp/definition", handleDefinition(ls)).Methods("GET")
	s.router.HandleFunc("/lsp/hover", handleHover(ls)).Methods("GET")
}

func (ls *LanguageServer) parseDocument(uri string, content string) error {
	file, err := parser.ParseFile(ls.fileSet, uri, content, parser.ParseComments)
	if err != nil {
		return fmt.Errorf("failed to parse document: %v", err)
	}

	symbols := ls.extractSymbols(file)

	ls.mu.Lock()
	defer ls.mu.Unlock()

	ls.documents[uri] = &Document{
		URI:     uri,
		Text:    content,
		AST:     file,
		Symbols: symbols,
		Version: ls.documents[uri].Version + 1,
	}

	return nil
}

func (ls *LanguageServer) extractSymbols(file *ast.File) []SymbolInfo {
	var symbols []SymbolInfo

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			pos := ls.fileSet.Position(node.Pos())
			end := ls.fileSet.Position(node.End())

			symbols = append(symbols, SymbolInfo{
				Name: node.Name.Name,
				Kind: "function",
				Location: Location{
					URI: file.Name.Name,
					Range: Range{
						Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
						End:   Position{Line: end.Line - 1, Character: end.Column - 1},
					},
				},
				Signature: ls.getFunctionSignature(node),
			})

		case *ast.TypeSpec:
			pos := ls.fileSet.Position(node.Pos())
			end := ls.fileSet.Position(node.End())

			symbols = append(symbols, SymbolInfo{
				Name: node.Name.Name,
				Kind: "type",
				Location: Location{
					URI: file.Name.Name,
					Range: Range{
						Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
						End:   Position{Line: end.Line - 1, Character: end.Column - 1},
					},
				},
			})
		}
		return true
	})

	return symbols
}

func (ls *LanguageServer) getFunctionSignature(fn *ast.FuncDecl) string {
	var params []string
	if fn.Type.Params != nil {
		for _, param := range fn.Type.Params.List {
			paramType := ls.nodeToString(param.Type)
			for _, name := range param.Names {
				params = append(params, fmt.Sprintf("%s %s", name.Name, paramType))
			}
		}
	}

	var returns []string
	if fn.Type.Results != nil {
		for _, result := range fn.Type.Results.List {
			resultType := ls.nodeToString(result.Type)
			if len(result.Names) > 0 {
				for _, name := range result.Names {
					returns = append(returns, fmt.Sprintf("%s %s", name.Name, resultType))
				}
			} else {
				returns = append(returns, resultType)
			}
		}
	}

	signature := fmt.Sprintf("func %s(%s)", fn.Name.Name, strings.Join(params, ", "))
	if len(returns) > 0 {
		if len(returns) == 1 {
			signature += " " + returns[0]
		} else {
			signature += fmt.Sprintf(" (%s)", strings.Join(returns, ", "))
		}
	}

	return signature
}

func (ls *LanguageServer) nodeToString(node ast.Node) string {
	switch n := node.(type) {
	case *ast.Ident:
		return n.Name
	case *ast.StarExpr:
		return "*" + ls.nodeToString(n.X)
	case *ast.ArrayType:
		return "[]" + ls.nodeToString(n.Elt)
	case *ast.SelectorExpr:
		return ls.nodeToString(n.X) + "." + n.Sel.Name
	default:
		return fmt.Sprintf("%T", node)
	}
}

// Handler functions
func handleOpenDocument(ls *LanguageServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var doc TextDocumentItem
		if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if err := ls.parseDocument(doc.URI, doc.Text); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "opened",
			"uri":    doc.URI,
		})
	}
}

func handleCloseDocument(ls *LanguageServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var doc TextDocumentItem
		if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		ls.mu.Lock()
		delete(ls.documents, doc.URI)
		ls.mu.Unlock()

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "closed",
			"uri":    doc.URI,
		})
	}
}

func handleChangeDocument(ls *LanguageServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var doc TextDocumentItem
		if err := json.NewDecoder(r.Body).Decode(&doc); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		if err := ls.parseDocument(doc.URI, doc.Text); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{
			"status": "changed",
			"uri":    doc.URI,
		})
	}
}

func handleDocumentSymbols(ls *LanguageServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		uri := r.URL.Query().Get("uri")
		if uri == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("uri parameter is required"))
			return
		}

		ls.mu.RLock()
		doc, exists := ls.documents[uri]
		ls.mu.RUnlock()

		if !exists {
			writeError(w, http.StatusNotFound, fmt.Errorf("document not found"))
			return
		}

		writeJSON(w, http.StatusOK, doc.Symbols)
	}
}

func handleCompletion(ls *LanguageServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Basic completion handler - can be extended based on needs
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"status": "not implemented",
		})
	}
}

func handleDefinition(ls *LanguageServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Basic definition handler - can be extended based on needs
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"status": "not implemented",
		})
	}
}

func handleHover(ls *LanguageServer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Basic hover handler - can be extended based on needs
		writeJSON(w, http.StatusNotImplemented, map[string]string{
			"status": "not implemented",
		})
	}
}

// Helper method for the Server struct
func (s *Server) GetWorkspaceRoot() string {
	// This should be configurable in your actual implementation
	return "."
}
