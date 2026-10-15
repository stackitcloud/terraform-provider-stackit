package tfclientcollection

import (
	"go/ast"
	"go/token"
	"go/types"

	"github.com/golangci/plugin-module-register/register"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

var Analyzer = &analysis.Analyzer{
	Name:     "tfclientcollection",
	Doc:      "Prevents the keeping the core.ClientCollection struct from being kept in a resource. Only clients should be extracted from it and kept.",
	Run:      run,
	Requires: []*analysis.Analyzer{inspect.Analyzer},
}

const (
	targetPkg    = "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/core"
	targetStruct = "ClientCollection"
)

func run(pass *analysis.Pass) (any, error) {
	inspector, ok := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	if !ok || inspector == nil {
		return nil, nil
	}

	nodeFilter := []ast.Node{
		(*ast.GenDecl)(nil),
		(*ast.StructType)(nil),
		(*ast.FuncType)(nil),
	}

	inspector.Preorder(nodeFilter, func(n ast.Node) {
		switch node := n.(type) {
		// Checks 'var' declarations
		case *ast.GenDecl:
			if node.Tok != token.VAR {
				return
			}
			for _, spec := range node.Specs {
				vspec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, name := range vspec.Names {
					obj := pass.TypesInfo.ObjectOf(name)
					if obj != nil && isForbiddenType(obj.Type()) {
						pass.Reportf(
							name.Pos(),
							"variable %q declared via 'var' uses forbidden type %s.%s",
							name.Name,
							targetPkg,
							targetStruct,
						)
					}
				}
			}

		// Checks struct fields
		case *ast.StructType:
			if node.Fields == nil {
				return
			}
			for _, field := range node.Fields.List {
				fieldType := pass.TypesInfo.TypeOf(field.Type)
				if isForbiddenType(fieldType) {
					pass.Reportf(
						field.Pos(),
						"struct field uses forbidden type %s.%s",
						targetPkg,
						targetStruct,
					)
				}
			}

		// Checks function, method, and function literal parameters
		case *ast.FuncType:
			if node.Params == nil {
				return
			}
			for _, param := range node.Params.List {
				paramType := pass.TypesInfo.TypeOf(param.Type)
				if isForbiddenType(paramType) {
					pass.Reportf(
						param.Pos(),
						"function parameter uses forbidden type %s.%s",
						targetPkg,
						targetStruct,
					)
				}
			}
		}
	})

	return nil, nil
}

func isForbiddenType(t types.Type) bool {
	if t == nil {
		return false
	}
	// Unwrap variadic parameters (...pkg.ForbiddenStruct -> pkg.ForbiddenStruct)
	if slice, ok := t.(*types.Slice); ok {
		t = slice.Elem()
	}
	// Unwrap pointers (*pkg.Struct -> pkg.Struct)
	if ptr, ok := t.(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := t.(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	if obj == nil || obj.Pkg() == nil {
		return false
	}
	return obj.Pkg().Path() == targetPkg && obj.Name() == targetStruct
}

func init() {
	register.Plugin("tfclientcollection", New)
}

func New(settings any) (register.LinterPlugin, error) {
	return &plugin{}, nil
}

type plugin struct{}

func (p *plugin) BuildAnalyzers() ([]*analysis.Analyzer, error) {
	return []*analysis.Analyzer{Analyzer}, nil
}

func (p *plugin) GetLoadMode() string {
	// LoadModeSyntax is required because we need to inspect the AST (Syntax trees)
	return register.LoadModeSyntax
}
