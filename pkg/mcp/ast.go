package mcp

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"
	_ "golang.org/x/tools/go/ast/astutil"
	"strings"
)

// ASTAnalyzer provides code analysis capabilities
type ASTAnalyzer struct {
	fileSet    *token.FileSet
	typeInfo   *types.Info
	packages   map[string]*ast.Package
	complexity map[string]int
}

// AnalysisResult contains the analysis output
type AnalysisResult struct {
	Imports     []ImportInfo    `json:"imports"`
	Functions   []FunctionInfo  `json:"functions"`
	Types       []TypeInfo      `json:"types"`
	Variables   []VariableInfo  `json:"variables"`
	References  []ReferenceInfo `json:"references"`
	Diagnostics []Diagnostic    `json:"diagnostics"`
	Metrics     CodeMetrics     `json:"metrics"`
}

type CodeMetrics struct {
	LinesOfCode     int `json:"lines_of_code"`
	CommentLines    int `json:"comment_lines"`
	FunctionCount   int `json:"function_count"`
	ComplexityScore int `json:"complexity_score"`
	InterfaceCount  int `json:"interface_count"`
	StructCount     int `json:"struct_count"`
	TestCount       int `json:"test_count"`
}

type Diagnostic struct {
	Severity string   `json:"severity"` // error, warning, info
	Message  string   `json:"message"`
	Location Location `json:"location"`
	Code     string   `json:"code"`
	Source   string   `json:"source"`
}

// NewASTAnalyzer creates a new AST analyzer
func NewASTAnalyzer(fset *token.FileSet) *ASTAnalyzer {
	return &ASTAnalyzer{
		fileSet: fset,
		typeInfo: &types.Info{
			Types:     make(map[ast.Expr]types.TypeAndValue),
			Defs:      make(map[*ast.Ident]types.Object),
			Uses:      make(map[*ast.Ident]types.Object),
			Implicits: make(map[ast.Node]types.Object),
		},
		packages:   make(map[string]*ast.Package),
		complexity: make(map[string]int),
	}
}

// AnalyzeFile performs deep analysis of a Go source file
func (a *ASTAnalyzer) AnalyzeFile(file *ast.File) (*AnalysisResult, error) {
	result := &AnalysisResult{
		Metrics: CodeMetrics{},
	}

	// Analyze imports
	result.Imports = a.analyzeImports(file)

	// Analyze functions
	result.Functions = a.analyzeFunctions(file)

	// Analyze types
	result.Types = a.analyzeTypes(file)

	// Analyze variables
	result.Variables = a.analyzeVariables(file)

	// Collect references
	result.References = a.analyzeReferences(file)

	// Calculate metrics
	result.Metrics = a.calculateMetrics(file)

	// Run diagnostics
	result.Diagnostics = a.runDiagnostics(file)

	return result, nil
}

func (a *ASTAnalyzer) analyzeImports(file *ast.File) []ImportInfo {
	var imports []ImportInfo
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		name := ""
		if imp.Name != nil {
			name = imp.Name.Name
		}

		imports = append(imports, ImportInfo{
			Path: path,
			Name: name,
			Used: a.isImportUsed(file, path),
		})
	}
	return imports
}

func (a *ASTAnalyzer) analyzeFunctions(file *ast.File) []FunctionInfo {
	var functions []FunctionInfo

	ast.Inspect(file, func(n ast.Node) bool {
		if fn, ok := n.(*ast.FuncDecl); ok {
			pos := a.fileSet.Position(fn.Pos())
			end := a.fileSet.Position(fn.End())

			doc := ""
			if fn.Doc != nil {
				doc = fn.Doc.Text()
			}

			complexity := a.calculateFunctionComplexity(fn)
			a.complexity[fn.Name.Name] = complexity

			functions = append(functions, FunctionInfo{
				Name:      fn.Name.Name,
				Signature: a.getFunctionSignature(fn),
				Doc:       doc,
				Location: Location{
					URI: file.Name.Name,
					Range: Range{
						Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
						End:   Position{Line: end.Line - 1, Character: end.Column - 1},
					},
				},
				Complexity: complexity,
			})
		}
		return true
	})

	return functions
}

func (a *ASTAnalyzer) analyzeTypes(file *ast.File) []TypeInfo {
	var types []TypeInfo

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeSpec:
			pos := a.fileSet.Position(node.Pos())
			end := a.fileSet.Position(node.End())

			typeInfo := TypeInfo{
				Name: node.Name.Name,
				Location: Location{
					URI: file.Name.Name,
					Range: Range{
						Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
						End:   Position{Line: end.Line - 1, Character: end.Column - 1},
					},
				},
			}

			// Handle different type kinds
			switch t := node.Type.(type) {
			case *ast.StructType:
				typeInfo.Kind = "struct"
				typeInfo.Fields = a.analyzeStructFields(t)
			case *ast.InterfaceType:
				typeInfo.Kind = "interface"
				typeInfo.Methods = a.analyzeInterfaceMethods(t)
			default:
				typeInfo.Kind = fmt.Sprintf("%T", node.Type)
			}

			types = append(types, typeInfo)
		}
		return true
	})

	return types
}

func (a *ASTAnalyzer) analyzeVariables(file *ast.File) []VariableInfo {
	var variables []VariableInfo

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.ValueSpec:
			pos := a.fileSet.Position(node.Pos())
			end := a.fileSet.Position(node.End())

			for _, name := range node.Names {
				variables = append(variables, VariableInfo{
					Name: name.Name,
					Type: a.getTypeString(node.Type),
					Location: Location{
						URI: file.Name.Name,
						Range: Range{
							Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
							End:   Position{Line: end.Line - 1, Character: end.Column - 1},
						},
					},
					Constant: node.Values != nil && len(node.Values) > 0,
				})
			}
		}
		return true
	})

	return variables
}

func (a *ASTAnalyzer) analyzeReferences(file *ast.File) []ReferenceInfo {
	var references []ReferenceInfo
	refMap := make(map[string]*ReferenceInfo)

	// First pass: collect all definitions
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			if obj := a.typeInfo.Defs[node]; obj != nil {
				pos := a.fileSet.Position(node.Pos())
				refMap[node.Name] = &ReferenceInfo{
					Name: node.Name,
					Kind: a.getIdentKind(obj),
					Location: Location{
						URI: file.Name.Name,
						Range: Range{
							Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
							End:   Position{Line: pos.Line - 1, Character: pos.Column - 1 + len(node.Name)},
						},
					},
					UsedAt: make([]Location, 0),
				}
			}
		}
		return true
	})

	// Second pass: collect all uses
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.Ident:
			if obj := a.typeInfo.Uses[node]; obj != nil {
				if ref, ok := refMap[obj.Name()]; ok {
					pos := a.fileSet.Position(node.Pos())
					ref.UsedAt = append(ref.UsedAt, Location{
						URI: file.Name.Name,
						Range: Range{
							Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
							End:   Position{Line: pos.Line - 1, Character: pos.Column - 1 + len(node.Name)},
						},
					})
				}
			}
		}
		return true
	})

	// Convert map to slice
	for _, ref := range refMap {
		references = append(references, *ref)
	}

	return references
}

func (a *ASTAnalyzer) calculateMetrics(file *ast.File) CodeMetrics {
	metrics := CodeMetrics{}

	// Count lines of code and comments
	lineCount := a.fileSet.Position(file.End()).Line
	metrics.LinesOfCode = lineCount

	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			metrics.FunctionCount++
			if strings.HasPrefix(node.Name.Name, "Test") {
				metrics.TestCount++
			}
		case *ast.TypeSpec:
			switch node.Type.(type) {
			case *ast.StructType:
				metrics.StructCount++
			case *ast.InterfaceType:
				metrics.InterfaceCount++
			}
		case *ast.CommentGroup:
			metrics.CommentLines += len(node.List)
		}
		return true
	})

	// Calculate total complexity
	for _, complexity := range a.complexity {
		metrics.ComplexityScore += complexity
	}

	return metrics
}

func (a *ASTAnalyzer) runDiagnostics(file *ast.File) []Diagnostic {
	var diagnostics []Diagnostic

	// Check for unused imports
	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if !a.isImportUsed(file, path) {
			pos := a.fileSet.Position(imp.Pos())
			diagnostics = append(diagnostics, Diagnostic{
				Severity: "warning",
				Message:  fmt.Sprintf("Unused import: %s", path),
				Location: Location{
					URI: file.Name.Name,
					Range: Range{
						Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
						End:   Position{Line: pos.Line - 1, Character: pos.Column - 1 + len(path)},
					},
				},
				Code:   "unused-import",
				Source: "go-analyzer",
			})
		}
	}

	// Check for exported symbols without documentation
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			if ast.IsExported(node.Name.Name) && node.Doc == nil {
				pos := a.fileSet.Position(node.Name.Pos())
				diagnostics = append(diagnostics, Diagnostic{
					Severity: "info",
					Message:  fmt.Sprintf("Exported function %s lacks documentation", node.Name.Name),
					Location: Location{
						URI: file.Name.Name,
						Range: Range{
							Start: Position{Line: pos.Line - 1, Character: pos.Column - 1},
							End:   Position{Line: pos.Line - 1, Character: pos.Column - 1 + len(node.Name.Name)},
						},
					},
					Code:   "missing-doc",
					Source: "go-analyzer",
				})
			}
		}
		return true
	})

	return diagnostics
}

// Helper functions

func (a *ASTAnalyzer) isImportUsed(file *ast.File, importPath string) bool {
	used := false
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok {
				if obj := a.typeInfo.Uses[ident]; obj != nil {
					if pkg, ok := obj.(*types.PkgName); ok {
						if pkg.Imported().Path() == importPath {
							used = true
							return false
						}
					}
				}
			}
		}
		return true
	})
	return used
}

func (a *ASTAnalyzer) calculateFunctionComplexity(fn *ast.FuncDecl) int {
	complexity := 1 // Base complexity

	ast.Inspect(fn, func(n ast.Node) bool {
		switch n.(type) {
		case *ast.IfStmt, *ast.ForStmt, *ast.RangeStmt, *ast.CaseClause,
			*ast.CommClause, *ast.BinaryExpr:
			complexity++
		}
		return true
	})

	return complexity
}

func (a *ASTAnalyzer) analyzeStructFields(structType *ast.StructType) []FieldInfo {
	var fields []FieldInfo

	for _, field := range structType.Fields.List {
		for _, name := range field.Names {
			var tags string
			if field.Tag != nil {
				tags = field.Tag.Value
			}

			fields = append(fields, FieldInfo{
				Name: name.Name,
				Type: a.getTypeString(field.Type),
				Doc:  field.Doc.Text(),
				Tags: tags,
			})
		}
	}

	return fields
}

func (a *ASTAnalyzer) analyzeInterfaceMethods(interfaceType *ast.InterfaceType) []MethodInfo {
	var methods []MethodInfo

	for _, method := range interfaceType.Methods.List {
		if len(method.Names) > 0 {
			methodType, ok := method.Type.(*ast.FuncType)
			if ok {
				methods = append(methods, MethodInfo{
					Name:      method.Names[0].Name,
					Signature: a.getFunctionTypeSignature(methodType),
					Doc:       method.Doc.Text(),
				})
			}
		}
	}

	return methods
}

func (a *ASTAnalyzer) getTypeString(expr ast.Expr) string {
	if expr == nil {
		return "unknown"
	}

	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + a.getTypeString(t.X)
	case *ast.ArrayType:
		return "[]" + a.getTypeString(t.Elt)
	case *ast.MapType:
		return fmt.Sprintf("map[%s]%s", a.getTypeString(t.Key), a.getTypeString(t.Value))
	case *ast.InterfaceType:
		return "interface{}"
	case *ast.SelectorExpr:
		return fmt.Sprintf("%s.%s", a.getTypeString(t.X), t.Sel.Name)
	case *ast.StructType:
		return "struct{...}"
	case *ast.FuncType:
		return a.getFunctionTypeSignature(t)
	case *ast.ChanType:
		switch t.Dir {
		case ast.SEND:
			return "chan<- " + a.getTypeString(t.Value)
		case ast.RECV:
			return "<-chan " + a.getTypeString(t.Value)
		default:
			return "chan " + a.getTypeString(t.Value)
		}
	default:
		return fmt.Sprintf("%T", expr)
	}
}

func (a *ASTAnalyzer) getFunctionSignature(fn *ast.FuncDecl) string {
	var receiver string
	if fn.Recv != nil && len(fn.Recv.List) > 0 {
		receiver = fmt.Sprintf("(%s) ", a.getTypeString(fn.Recv.List[0].Type))
	}

	return fmt.Sprintf("func %s%s%s", receiver, fn.Name.Name, a.getFunctionTypeSignature(fn.Type))
}

func (a *ASTAnalyzer) getFunctionTypeSignature(fnType *ast.FuncType) string {
	var params []string
	if fnType.Params != nil {
		for _, p := range fnType.Params.List {
			paramType := a.getTypeString(p.Type)
			if len(p.Names) == 0 {
				params = append(params, paramType)
			} else {
				for _, name := range p.Names {
					params = append(params, fmt.Sprintf("%s %s", name.Name, paramType))
				}
			}
		}
	}

	var returns []string
	if fnType.Results != nil {
		for _, r := range fnType.Results.List {
			returnType := a.getTypeString(r.Type)
			if len(r.Names) == 0 {
				returns = append(returns, returnType)
			} else {
				for _, name := range r.Names {
					returns = append(returns, fmt.Sprintf("%s %s", name.Name, returnType))
				}
			}
		}
	}

	var returnStr string
	if len(returns) == 0 {
		returnStr = ""
	} else if len(returns) == 1 {
		returnStr = " " + returns[0]
	} else {
		returnStr = " (" + strings.Join(returns, ", ") + ")"
	}

	return fmt.Sprintf("(%s)%s", strings.Join(params, ", "), returnStr)
}

func (a *ASTAnalyzer) getIdentKind(obj types.Object) string {
	switch obj.(type) {
	case *types.Func:
		return "function"
	case *types.Var:
		return "variable"
	case *types.Const:
		return "constant"
	case *types.TypeName:
		return "type"
	case *types.PkgName:
		return "package"
	case *types.Label:
		return "label"
	case *types.Builtin:
		return "builtin"
	case *types.Nil:
		return "nil"
	default:
		return "unknown"
	}
}

// Additional analysis methods

func (a *ASTAnalyzer) AnalyzePackage(pkgPath string, files []*ast.File) (*AnalysisResult, error) {
	pkg := &ast.Package{
		Name:  files[0].Name.Name,
		Files: make(map[string]*ast.File),
	}

	for _, file := range files {
		pkg.Files[a.fileSet.Position(file.Pos()).Filename] = file
	}

	a.packages[pkgPath] = pkg

	// Combine results from all files
	result := &AnalysisResult{
		Imports:   make([]ImportInfo, 0),
		Functions: make([]FunctionInfo, 0),
		Types:     make([]TypeInfo, 0),
		Variables: make([]VariableInfo, 0),
		Metrics:   CodeMetrics{},
	}

	for _, file := range files {
		fileResult, err := a.AnalyzeFile(file)
		if err != nil {
			return nil, err
		}

		result.Imports = append(result.Imports, fileResult.Imports...)
		result.Functions = append(result.Functions, fileResult.Functions...)
		result.Types = append(result.Types, fileResult.Types...)
		result.Variables = append(result.Variables, fileResult.Variables...)
		result.Diagnostics = append(result.Diagnostics, fileResult.Diagnostics...)

		// Aggregate metrics
		result.Metrics.LinesOfCode += fileResult.Metrics.LinesOfCode
		result.Metrics.CommentLines += fileResult.Metrics.CommentLines
		result.Metrics.FunctionCount += fileResult.Metrics.FunctionCount
		result.Metrics.ComplexityScore += fileResult.Metrics.ComplexityScore
		result.Metrics.InterfaceCount += fileResult.Metrics.InterfaceCount
		result.Metrics.StructCount += fileResult.Metrics.StructCount
		result.Metrics.TestCount += fileResult.Metrics.TestCount
	}

	return result, nil
}

// AnalyzeDependencies analyzes package dependencies
func (a *ASTAnalyzer) AnalyzeDependencies(file *ast.File) map[string][]string {
	deps := make(map[string][]string)

	for _, imp := range file.Imports {
		path := strings.Trim(imp.Path.Value, "\"")
		if a.isImportUsed(file, path) {
			pkg := file.Name.Name
			if deps[pkg] == nil {
				deps[pkg] = make([]string, 0)
			}
			deps[pkg] = append(deps[pkg], path)
		}
	}

	return deps
}
