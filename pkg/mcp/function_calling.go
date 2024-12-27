package mcp

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"sync"
)

// FunctionHandler manages function registration and execution
type FunctionHandler struct {
	functions map[string]interface{}
	mu        sync.RWMutex
}

// FunctionMetadata represents metadata about a registered function
type FunctionMetadata struct {
	Name       string         `json:"name"`
	Arguments  []ArgumentInfo `json:"arguments"`
	ReturnType string         `json:"return_type"`
}

// ArgumentInfo represents information about a function argument
type ArgumentInfo struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Required bool   `json:"required"`
}

// FunctionRequest represents a function call request
type FunctionRequest struct {
	Name      string        `json:"name"`
	Arguments []interface{} `json:"arguments"`
}

// NewFunctionHandler creates a new function handler instance
func NewFunctionHandler() *FunctionHandler {
	return &FunctionHandler{
		functions: make(map[string]interface{}),
	}
}

// RegisterFunction registers a function with the handler
func (h *FunctionHandler) RegisterFunction(name string, fn interface{}) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	if _, exists := h.functions[name]; exists {
		return fmt.Errorf("function %s is already registered", name)
	}

	fnType := reflect.TypeOf(fn)
	if fnType.Kind() != reflect.Func {
		return fmt.Errorf("provided value must be a function")
	}

	h.functions[name] = fn
	return nil
}

// GetFunctionMetadata returns metadata for all registered functions
func (h *FunctionHandler) GetFunctionMetadata() []FunctionMetadata {
	h.mu.RLock()
	defer h.mu.RUnlock()

	metadata := make([]FunctionMetadata, 0, len(h.functions))
	for name, fn := range h.functions {
		fnType := reflect.TypeOf(fn)
		args := make([]ArgumentInfo, fnType.NumIn())

		for i := 0; i < fnType.NumIn(); i++ {
			argType := fnType.In(i)
			args[i] = ArgumentInfo{
				Name:     fmt.Sprintf("arg%d", i),
				Type:     argType.String(),
				Required: true,
			}
		}

		returnType := "void"
		if fnType.NumOut() > 0 {
			returnType = fnType.Out(0).String()
		}

		metadata = append(metadata, FunctionMetadata{
			Name:       name,
			Arguments:  args,
			ReturnType: returnType,
		})
	}

	return metadata
}

// AddFunctionHandler adds function handling capabilities to the MCP server
func (s *Server) AddFunctionHandler() {
	handler := NewFunctionHandler()

	// Add example built-in functions
	handler.RegisterFunction("echo", func(msg string) string { return msg })

	// Register routes
	s.router.HandleFunc("/function/list", handleListFunctions(handler)).Methods("GET")
	s.router.HandleFunc("/function/call", handleCallFunction(handler)).Methods("POST")
}

func handleListFunctions(h *FunctionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		metadata := h.GetFunctionMetadata()
		writeJSON(w, http.StatusOK, metadata)
	}
}

func handleCallFunction(h *FunctionHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req FunctionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		h.mu.RLock()
		fn, exists := h.functions[req.Name]
		h.mu.RUnlock()

		if !exists {
			writeError(w, http.StatusNotFound, fmt.Errorf("function %s not found", req.Name))
			return
		}

		fnValue := reflect.ValueOf(fn)
		fnType := fnValue.Type()

		if len(req.Arguments) != fnType.NumIn() {
			writeError(w, http.StatusBadRequest, fmt.Errorf("expected %d arguments, got %d", fnType.NumIn(), len(req.Arguments)))
			return
		}

		args := make([]reflect.Value, len(req.Arguments))
		for i, arg := range req.Arguments {
			expectedType := fnType.In(i)
			argValue := reflect.ValueOf(arg)

			// Handle type conversion
			if !argValue.Type().AssignableTo(expectedType) {
				convertedArg, err := convertArgument(arg, expectedType)
				if err != nil {
					writeError(w, http.StatusBadRequest, fmt.Errorf("invalid argument %d: %v", i, err))
					return
				}
				args[i] = convertedArg
			} else {
				args[i] = argValue
			}
		}

		results := fnValue.Call(args)
		response := make(map[string]interface{})

		if len(results) > 0 {
			response["result"] = results[0].Interface()
		} else {
			response["result"] = nil
		}

		writeJSON(w, http.StatusOK, response)
	}
}

// convertArgument attempts to convert an argument to the expected type
func convertArgument(arg interface{}, expectedType reflect.Type) (reflect.Value, error) {
	argValue := reflect.ValueOf(arg)

	// Handle numeric type conversions
	switch expectedType.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if v, ok := arg.(float64); ok {
			return reflect.ValueOf(int64(v)).Convert(expectedType), nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if v, ok := arg.(float64); ok {
			return reflect.ValueOf(uint64(v)).Convert(expectedType), nil
		}
	case reflect.Float32, reflect.Float64:
		if v, ok := arg.(float64); ok {
			return reflect.ValueOf(v).Convert(expectedType), nil
		}
	}

	if !argValue.Type().ConvertibleTo(expectedType) {
		return reflect.Value{}, fmt.Errorf("cannot convert %v to %v", argValue.Type(), expectedType)
	}

	return argValue.Convert(expectedType), nil
}
