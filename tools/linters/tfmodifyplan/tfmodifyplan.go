package tfmodifyplan

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"

	"github.com/golangci/plugin-module-register/register"
)

var Analyzer = &analysis.Analyzer{
	Name:     "tfmodifyplan",
	Doc:      "Ensures every resource with a region field implements the ModifyPlan method.",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

const (
	targetPkgName  = "github.com/stackitcloud/terraform-provider-stackit/stackit/internal/utils"
	targetFuncName = "AdaptRegion"
)

func run(pass *analysis.Pass) (interface{}, error) {
	ins := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)

	// structData keeps track of methods attached to a struct receiver
	type structData struct {
		schemaMethod     *ast.FuncDecl
		modifyPlanMethod *ast.FuncDecl
		isResource       bool // True if it has Create, Update, or Delete methods
	}

	resources := make(map[string]*structData)

	// Filter down to only function declarations
	nodeFilter := []ast.Node{
		(*ast.FuncDecl)(nil),
	}

	// Pass 1: Collect and group all methods by their receiver struct
	ins.Preorder(nodeFilter, func(n ast.Node) {
		fn := n.(*ast.FuncDecl)

		// Skip if it doesn't have a receiver (not a struct method)
		if fn.Recv == nil || len(fn.Recv.List) == 0 {
			return
		}

		// Extract receiver struct name
		var recvName string
		switch t := fn.Recv.List[0].Type.(type) {
		case *ast.StarExpr: // Pointer receiver: *MyResource
			if ident, ok := t.X.(*ast.Ident); ok {
				recvName = ident.Name
			}
		case *ast.Ident: // Value receiver: MyResource
			recvName = t.Name
		}

		if recvName == "" {
			return
		}

		if resources[recvName] == nil {
			resources[recvName] = &structData{}
		}

		// Identify the role of the method
		switch fn.Name.Name {
		case "Schema":
			resources[recvName].schemaMethod = fn
		case "ModifyPlan":
			resources[recvName].modifyPlanMethod = fn
		case "Create", "Update", "Delete":
			// Data sources do not have Create/Update/Delete in the TF Plugin Framework.
			// This heuristic guarantees we are looking at a Resource.
			resources[recvName].isResource = true
		}
	})

	// Pass 2: Analyze the collected data against your business logic rules
	for recvName, data := range resources {
		// Only analyze valid TF Resources that have a Schema method
		if !data.isResource || data.schemaMethod == nil {
			continue
		}

		// Check if the string "region" is defined anywhere in the Schema method
		hasRegion := false
		ast.Inspect(data.schemaMethod, func(n ast.Node) bool {
			if lit, ok := n.(*ast.BasicLit); ok {
				if lit.Kind == token.STRING && lit.Value == `"region"` {
					hasRegion = true
					return false // Found it, stop walking this branch
				}
			}
			return true
		})

		// If it doesn't have a region attribute, skip it.
		if !hasRegion {
			continue
		}

		// Has region, but no ModifyPlan method
		if data.modifyPlanMethod == nil {
			pass.Reportf(
				data.schemaMethod.Pos(),
				"Terraform resource '%s' defines a 'region' field but does not implement the ModifyPlan method.",
				recvName,
			)
			continue
		}

		// Check if the specific function is called inside ModifyPlan
		hasRequiredFuncCall := false
		ast.Inspect(data.modifyPlanMethod, func(n ast.Node) bool {
			if call, ok := n.(*ast.CallExpr); ok {
				if sel, ok := call.Fun.(*ast.SelectorExpr); ok {
					// Check if the method being called matches our target function name
					if sel.Sel.Name == targetFuncName {
						if ident, ok := sel.X.(*ast.Ident); ok {
							// Use type info to resolve the identifier to its actual package
							obj := pass.TypesInfo.Uses[ident]
							if pkgName, ok := obj.(*types.PkgName); ok {
								// Compare the actual import path
								if pkgName.Imported().Path() == targetPkgName {
									hasRequiredFuncCall = true
									return false // Found it, stop walking this branch
								}
							}
						}
					}
				}
			}
			return true
		})

		// Has ModifyPlan, but missing the required package/function call
		if !hasRequiredFuncCall {
			pass.Reportf(
				data.modifyPlanMethod.Pos(),
				"Terraform resource '%s' ModifyPlan method must call %s.%s().",
				recvName,
				targetPkgName,
				targetFuncName,
			)
		}
	}

	return nil, nil
}

func init() {
	register.Plugin("tfmodifyplan", New)
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
