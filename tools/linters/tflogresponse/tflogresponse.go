package tflogresponse

import (
	"go/ast"
	"strings"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
)

var Analyzer = &analysis.Analyzer{
	Name: "tflogresponse",
	Doc:  "Ensures that ctx = core.LogResponse(ctx) is called in every resource/datasource CRUD method.",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			// Look for function declarations
			fn, ok := n.(*ast.FuncDecl)
			if !ok || fn.Recv == nil {
				return true // We only care about methods (which have receivers)
			}

			// Filter by Terraform CRUD method names
			name := fn.Name.Name
			if name != "Create" && name != "Read" && name != "Update" && name != "Delete" {
				return true
			}

			// Ensure the method has exactly 3 parameters (e.g., ctx, req, resp)
			if fn.Type.Params == nil || len(fn.Type.Params.List) != 3 {
				return true
			}

			// Extract the name of the context variable (usually "ctx")
			if len(fn.Type.Params.List[0].Names) == 0 {
				return true
			}
			ctxName := fn.Type.Params.List[0].Names[0].Name

			// Verify it's actually a Terraform framework method by checking if 
			// param 2 ends with "Request" and param 3 ends with "Response"
			if !hasSuffixType(fn.Type.Params.List[1], "Request") || !hasSuffixType(fn.Type.Params.List[2], "Response") {
				return true
			}

			// Scan the body for: ctxName = core.LogResponse(ctxName)
			found := false
			if fn.Body != nil {
				for _, stmt := range fn.Body.List {
					// We are looking for an assignment statement: A = B
					assign, ok := stmt.(*ast.AssignStmt)
					if !ok || len(assign.Lhs) != 1 || len(assign.Rhs) != 1 {
						continue
					}

					// Left side must be the context variable
					lhsIdent, ok := assign.Lhs[0].(*ast.Ident)
					if !ok || lhsIdent.Name != ctxName {
						continue
					}

					// Right side must be a function call
					call, ok := assign.Rhs[0].(*ast.CallExpr)
					if !ok {
						continue
					}

					// The function called must be core.LogResponse
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok || sel.Sel.Name != "LogResponse" {
						continue
					}

					pkgIdent, ok := sel.X.(*ast.Ident)
					if !ok || pkgIdent.Name != "core" {
						continue
					}

					// The argument passed must be the context variable
					if len(call.Args) == 1 {
						if argIdent, ok := call.Args[0].(*ast.Ident); ok && argIdent.Name == ctxName {
							found = true
							break
						}
					}
				}
			}

			// If the exact assignment wasn't found in the function body, report it
			if !found {
				pass.Reportf(
					fn.Pos(), 
					"Terraform %s method must call %s = core.LogResponse(%s)", 
					name, ctxName, ctxName,
				)
			}

			return true
		})
	}
	return nil, nil
}

// Helper to roughly check if a parameter type ends with a specific string (like "Request" or "Response")
func hasSuffixType(field *ast.Field, suffix string) bool {
	var typeName string
	switch t := field.Type.(type) {
	case *ast.SelectorExpr:
		typeName = t.Sel.Name // e.g., resource.CreateRequest
	case *ast.StarExpr:
		if sel, ok := t.X.(*ast.SelectorExpr); ok {
			typeName = sel.Sel.Name // e.g., *resource.CreateResponse
		}
	}
	return strings.HasSuffix(typeName, suffix)
}

func init() {
	register.Plugin("tflogresponse", New)
}

func New(settings any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

func (p *plugin) GetLoadMode() string {
	return register.LoadModeSyntax
}
