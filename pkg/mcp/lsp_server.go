package mcp

import (
	"encoding/json"
	"go/parser"
	"go/token"
	"net/http"
)

// AnalysisRequest represents a request for code analysis
type AnalysisRequest struct {
	URI     string `json:"uri"`
	Content string `json:"content"`
}

// AddAnalysisHandler adds code analysis endpoints to the MCP server
func (s *Server) AddAnalysisHandler() {
	// Create analyzers
	fset := token.NewFileSet()
	analyzer := NewASTAnalyzer(fset)

	// Register analysis endpoints
	s.router.HandleFunc("/analyze/file", handleFileAnalysis(analyzer)).Methods("POST")
	s.router.HandleFunc("/analyze/dependencies", handleDependencyAnalysis(analyzer)).Methods("POST")
	s.router.HandleFunc("/analyze/metrics", handleMetricsAnalysis(analyzer)).Methods("POST")
}

func handleFileAnalysis(analyzer *ASTAnalyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AnalysisRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		// Parse the file
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, req.URI, req.Content, parser.ParseComments)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		// Analyze the file
		result, err := analyzer.AnalyzeFile(file)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, result)
	}
}

func handleDependencyAnalysis(analyzer *ASTAnalyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AnalysisRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		// Parse the file
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, req.URI, req.Content, parser.ParseComments)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		// Analyze dependencies
		deps := analyzer.AnalyzeDependencies(file)

		writeJSON(w, http.StatusOK, deps)
	}
}

func handleMetricsAnalysis(analyzer *ASTAnalyzer) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req AnalysisRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		// Parse the file
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, req.URI, req.Content, parser.ParseComments)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		// Get metrics
		result, err := analyzer.AnalyzeFile(file)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}

		writeJSON(w, http.StatusOK, result.Metrics)
	}
}

// Update the Server struct to integrate the analysis capabilities
func (s *Server) InitializeAnalysis() {
	s.AddAnalysisHandler()
}

// Add documentation endpoints
func (s *Server) AddDocumentationEndpoints() {
	s.router.HandleFunc("/docs/analysis", func(w http.ResponseWriter, r *http.Request) {
		docs := map[string]interface{}{
			"endpoints": []map[string]string{
				{
					"path":        "/analyze/file",
					"method":      "POST",
					"description": "Performs complete analysis of a Go source file",
				},
				{
					"path":        "/analyze/dependencies",
					"method":      "POST",
					"description": "Analyzes package dependencies",
				},
				{
					"path":        "/analyze/metrics",
					"method":      "POST",
					"description": "Retrieves code metrics",
				},
			},
			"requestFormat": AnalysisRequest{
				URI:     "path/to/file.go",
				Content: "source code content",
			},
		}
		writeJSON(w, http.StatusOK, docs)
	}).Methods("GET")
}
