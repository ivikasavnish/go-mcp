package mcp

// ImportInfo represents information about an import declaration
type ImportInfo struct {
	Path string `json:"path"` // Import path
	Name string `json:"name"` // Local name (alias) if any
	Used bool   `json:"used"` // Whether the import is used in the code
}

// FunctionInfo represents information about a function declaration
type FunctionInfo struct {
	Name       string          `json:"name"`       // Function name
	Signature  string          `json:"signature"`  // Function signature
	Doc        string          `json:"doc"`        // Documentation comments
	Location   Location        `json:"location"`   // Position in source
	Complexity int             `json:"complexity"` // Cyclomatic complexity
	IsMethod   bool            `json:"is_method"`  // Whether it's a method
	Receiver   string          `json:"receiver"`   // Receiver type if method
	Parameters []ParameterInfo `json:"parameters"` // Function parameters
	Returns    []ParameterInfo `json:"returns"`    // Return values
}

// ParameterInfo represents a function parameter or return value
type ParameterInfo struct {
	Name       string `json:"name"`     // Parameter name
	Type       string `json:"type"`     // Parameter type
	IsVariadic bool   `json:"variadic"` // Whether it's a variadic parameter
}

// TypeInfo represents information about a type declaration
type TypeInfo struct {
	Name       string       `json:"name"`       // Type name
	Kind       string       `json:"kind"`       // Type kind (struct, interface, etc.)
	Doc        string       `json:"doc"`        // Documentation comments
	Location   Location     `json:"location"`   // Position in source
	Fields     []FieldInfo  `json:"fields"`     // Fields for structs
	Methods    []MethodInfo `json:"methods"`    // Methods for types
	Implements []string     `json:"implements"` // Interfaces this type implements
}

// FieldInfo represents a struct field
type FieldInfo struct {
	Name    string `json:"name"`  // Field name
	Type    string `json:"type"`  // Field type
	Doc     string `json:"doc"`   // Field documentation
	Tags    string `json:"tags"`  // Field tags
	IsEmbed bool   `json:"embed"` // Whether it's an embedded field
}

// MethodInfo represents a method declaration
type MethodInfo struct {
	Name       string          `json:"name"`       // Method name
	Signature  string          `json:"signature"`  // Method signature
	Doc        string          `json:"doc"`        // Documentation
	Parameters []ParameterInfo `json:"parameters"` // Method parameters
	Returns    []ParameterInfo `json:"returns"`    // Return values
}

// VariableInfo represents information about a variable declaration
type VariableInfo struct {
	Name     string   `json:"name"`     // Variable name
	Type     string   `json:"type"`     // Variable type
	Location Location `json:"location"` // Position in source
	Constant bool     `json:"constant"` // Whether it's a constant
	Value    string   `json:"value"`    // Initial value if constant
	Doc      string   `json:"doc"`      // Documentation comments
	Scope    string   `json:"scope"`    // Variable scope (package, local)
}

// ReferenceInfo represents information about symbol references
type ReferenceInfo struct {
	Name     string     `json:"name"`     // Symbol name
	Kind     string     `json:"kind"`     // Symbol kind (variable, function, etc.)
	Location Location   `json:"location"` // Definition location
	UsedAt   []Location `json:"used_at"`  // Usage locations
	Scope    string     `json:"scope"`    // Reference scope
}
